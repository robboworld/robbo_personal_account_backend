package usecase

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
)

// kidGatewayFake keeps kid accounts in memory on top of the ЛК roster of localGateway.
type kidGatewayFake struct {
	*localGateway
	rooms   map[string]*models.ClassKidLoginRoomDB
	kids    map[string]*models.ClassKidLoginDB
	history []models.ClassHistoryEventDB
}

func newKidGatewayFake() *kidGatewayFake {
	return &kidGatewayFake{
		localGateway: newLocalGateway(),
		rooms:        map[string]*models.ClassKidLoginRoomDB{},
		kids:         map[string]*models.ClassKidLoginDB{},
	}
}

func (g *kidGatewayFake) AddHistoryEvent(e *models.ClassHistoryEventDB) error {
	g.history = append(g.history, *e)
	return nil
}

func (g *kidGatewayFake) actions() []string {
	out := make([]string, 0, len(g.history))
	for _, e := range g.history {
		out = append(out, e.Action)
	}
	return out
}

func (g *kidGatewayFake) GetKidRoom(classID string) (*models.ClassKidLoginRoomDB, error) {
	if r, ok := g.rooms[classID]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, teacherclass.ErrNotFound
}

func (g *kidGatewayFake) GetKidRoomBySlug(slug string) (*models.ClassKidLoginRoomDB, error) {
	for _, r := range g.rooms {
		if r.LoginSlug == slug {
			cp := *r
			return &cp, nil
		}
	}
	return nil, teacherclass.ErrNotFound
}

func (g *kidGatewayFake) GetKidRoomByCode(code string) (*models.ClassKidLoginRoomDB, error) {
	for _, r := range g.rooms {
		if r.RoomCode != nil && *r.RoomCode == code {
			cp := *r
			return &cp, nil
		}
	}
	return nil, teacherclass.ErrNotFound
}

// SaveKidRoom keeps the unique index on room_code, as in Postgres.
func (g *kidGatewayFake) SaveKidRoom(room *models.ClassKidLoginRoomDB) error {
	if room.RoomCode != nil {
		for id, r := range g.rooms {
			if id != room.ClassID && r.RoomCode != nil && *r.RoomCode == *room.RoomCode {
				return teacherclass.ErrConflict
			}
		}
	}
	cp := *room
	g.rooms[room.ClassID] = &cp
	return nil
}

func (g *kidGatewayFake) CreateKidLogin(row *models.ClassKidLoginDB) error {
	cp := *row
	g.kids[row.UserID] = &cp
	return nil
}

func (g *kidGatewayFake) ListKidLogins(classID string) ([]models.ClassKidLoginDB, error) {
	var out []models.ClassKidLoginDB
	for _, k := range g.kids {
		if k.ClassID == classID {
			out = append(out, *k)
		}
	}
	return out, nil
}

func (g *kidGatewayFake) CountKidLogins(classID string) (int64, error) {
	rows, _ := g.ListKidLogins(classID)
	return int64(len(rows)), nil
}

func (g *kidGatewayFake) GetKidLogin(userID string) (*models.ClassKidLoginDB, error) {
	if k, ok := g.kids[userID]; ok {
		cp := *k
		return &cp, nil
	}
	return nil, teacherclass.ErrNotFound
}

func (g *kidGatewayFake) GetKidLoginByCardHash(hash string) (*models.ClassKidLoginDB, error) {
	for _, k := range g.kids {
		if k.CardTokenHash != nil && *k.CardTokenHash == hash {
			cp := *k
			return &cp, nil
		}
	}
	return nil, teacherclass.ErrNotFound
}

func (g *kidGatewayFake) SetKidPictures(userID string, pictures []int) error {
	k := g.kids[userID]
	k.Pictures = pq.Int64Array(picturesArray(pictures))
	k.FailedAttempts = 0
	k.LockedAt = nil
	return nil
}

func (g *kidGatewayFake) SetKidCardHash(userID, hash string) error {
	h := hash
	g.kids[userID].CardTokenHash = &h
	return nil
}

func (g *kidGatewayFake) UnlockKid(userID string) error {
	g.kids[userID].FailedAttempts = 0
	g.kids[userID].LockedAt = nil
	return nil
}

func (g *kidGatewayFake) RecordKidLoginFailure(userID string, lockAfter int) (bool, error) {
	k := g.kids[userID]
	k.FailedAttempts++
	if k.LockedAt == nil && k.FailedAttempts >= lockAfter {
		now := time.Now()
		k.LockedAt = &now
	}
	return k.LockedAt != nil, nil
}

func (g *kidGatewayFake) RecordKidLoginSuccess(userID string, at time.Time) error {
	g.kids[userID].FailedAttempts = 0
	g.kids[userID].LastLoginAt = &at
	return nil
}

type kidLMS struct {
	nextID   int64
	created  []string // usernames
	emails   []string
	inactive map[string]bool
}

// withKidLMS stands in for LMS MySQL: users are made in memory from id 1000.
func withKidLMS(t *testing.T) *kidLMS {
	t.Helper()
	lms := &kidLMS{nextID: 1000, inactive: map[string]bool{}}
	prevCreate, prevLookup, prevNow := createKidLMSUser, lookupKidLMSUser, kidNow
	createKidLMSUser = func(username, email, name string) (int64, error) {
		lms.nextID++
		lms.created = append(lms.created, username)
		lms.emails = append(lms.emails, email)
		return lms.nextID, nil
	}
	lookupKidLMSUser = func(userID string) (*kidLMSUser, error) {
		return &kidLMSUser{Username: "k-" + userID, Email: userID + "@kids.robbo.invalid", Active: !lms.inactive[userID]}, nil
	}
	t.Cleanup(func() { createKidLMSUser, lookupKidLMSUser, kidNow = prevCreate, prevLookup, prevNow })
	return lms
}

// wrongSecret is the secret with its last picture changed.
func wrongSecret(pictures []int) []int {
	out := append([]int(nil), pictures...)
	out[len(out)-1] = (out[len(out)-1] + 1) % models.KidPictureCount
	return out
}

func TestRandomPicturesAreFiveFromTheGrid(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		p, err := randomPictures()
		if err != nil || len(p) != models.KidSecretLength {
			t.Fatalf("randomPictures = %v, %v", p, err)
		}
		for _, v := range p {
			if v < 0 || v >= models.KidPictureCount {
				t.Fatalf("picture %d out of range", v)
			}
			seen[v] = true
		}
	}
	if len(seen) != models.KidPictureCount {
		t.Fatalf("200 secrets used %d of %d pictures", len(seen), models.KidPictureCount)
	}
}

func setKidClock(at time.Time) { kidNow = func() time.Time { return at } }

var kidTestStart = time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)

func newKidUseCase(t *testing.T) (*UseCaseImpl, *kidGatewayFake, *kidLMS) {
	t.Helper()
	withoutLMSNames(t)
	lms := withKidLMS(t)
	setKidClock(kidTestStart)
	gw := newKidGatewayFake()
	gw.invites["c1"] = &models.ClassInviteDB{ID: "c1", OwnerTeacherID: "10", DisplayName: "2Б", InviteSlug: "s1", InviteCode: "AAAA-BBBB", IsActive: true}
	gw.invites["c2"] = &models.ClassInviteDB{ID: "c2", OwnerTeacherID: "10", DisplayName: "3А", InviteSlug: "s2", InviteCode: "CCCC-DDDD", IsActive: true}
	gw.invites["lms"] = &models.ClassInviteDB{ID: "lms", OwnerTeacherID: "10", DisplayName: "LMS", LmsLinked: true, IsActive: true}
	uc := &UseCaseImpl{gw: gw, kids: gw}
	return uc, gw, lms
}

func createTwoKids(t *testing.T, uc *UseCaseImpl) []teacherclass.KidDTO {
	t.Helper()
	kids, err := uc.CreateKids("10", models.Teacher, "c1", []string{"  Маша   К. ", "", "Петя С."}, true)
	if err != nil {
		t.Fatalf("CreateKids: %v", err)
	}
	if len(kids) != 2 {
		t.Fatalf("kids = %d, want 2", len(kids))
	}
	return kids
}

func openLesson(t *testing.T, uc *UseCaseImpl, gw *kidGatewayFake) string {
	t.Helper()
	if _, err := uc.OpenKidLogin("10", models.Teacher, "c1", 90); err != nil {
		t.Fatalf("OpenKidLogin: %v", err)
	}
	return gw.rooms["c1"].LoginSlug
}

func TestCreateKidsMakesLMSUsersOnTheRoster(t *testing.T) {
	uc, gw, lms := newKidUseCase(t)
	kids := createTwoKids(t, uc)

	if kids[0].Name != "Маша К." {
		t.Fatalf("name = %q, want spaces collapsed", kids[0].Name)
	}
	if kids[0].Avatar == kids[1].Avatar {
		t.Fatalf("two kids share avatar %d", kids[0].Avatar)
	}
	for i, k := range kids {
		if len(k.Pictures) != models.KidSecretLength {
			t.Fatalf("kid %d pictures = %v, want %d", i, k.Pictures, models.KidSecretLength)
		}
		for _, p := range k.Pictures {
			if p < 0 || p >= models.KidPictureCount {
				t.Fatalf("picture %d out of range", p)
			}
		}
		if !gw.members["c1"][k.UserID] {
			t.Fatalf("kid %s not on the roster", k.UserID)
		}
	}
	for i, u := range lms.created {
		if !strings.HasPrefix(u, "k-") || len(u) != 12 {
			t.Fatalf("username %q: want k- and 10 random characters", u)
		}
		if lms.emails[i] != u+"@kids.robbo.invalid" {
			t.Fatalf("email %q: want the .invalid domain", lms.emails[i])
		}
	}
	if got := gw.actions(); len(got) != 1 || got[0] != models.HistoryKidsCreated || gw.history[0].Detail != "2" {
		t.Fatalf("history = %v, want one kids_created for 2", got)
	}
	if gw.rooms["c1"] == nil || len(gw.rooms["c1"].LoginSlug) != kidLoginSlugLen {
		t.Fatal("the class page was not made")
	}
}

func TestCreateKidsRefusals(t *testing.T) {
	uc, _, _ := newKidUseCase(t)
	if _, err := uc.CreateKids("10", models.Teacher, "c1", []string{"Маша"}, false); !errors.Is(err, teacherclass.ErrKidConsentRequired) {
		t.Fatalf("no consent: %v", err)
	}
	if _, err := uc.CreateKids("10", models.Teacher, "lms", []string{"Маша"}, true); !errors.Is(err, teacherclass.ErrLmsLinkedClass) {
		t.Fatalf("LMS class: %v", err)
	}
	if _, err := uc.CreateKids("99", models.Teacher, "c1", []string{"Маша"}, true); !errors.Is(err, teacherclass.ErrForbidden) {
		t.Fatalf("stranger: %v", err)
	}
	if _, err := uc.CreateKids("10", models.Teacher, "c1", []string{strings.Repeat("я", 41)}, true); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("long name: %v", err)
	}
	if _, err := uc.CreateKids("10", models.Teacher, "c1", []string{" ", ""}, true); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("no names: %v", err)
	}
	many := make([]string, 41)
	for i := range many {
		many[i] = "Ученик " + strconv.Itoa(i)
	}
	if _, err := uc.CreateKids("10", models.Teacher, "c1", many, true); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("41 at once: %v", err)
	}
	if _, err := uc.CreateKids("10", models.Teacher, "c1", many[:40], true); err != nil {
		t.Fatalf("40: %v", err)
	}
	if _, err := uc.CreateKids("10", models.Teacher, "c1", many[:21], true); !errors.Is(err, teacherclass.ErrTooManyKids) {
		t.Fatalf("61 in class: %v", err)
	}
}

func TestKidLoginPageListsNamesOnlyDuringTheLesson(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	createTwoKids(t, uc)
	if _, err := uc.ListKids("10", models.Teacher, "c1"); err != nil {
		t.Fatal(err)
	}
	slug := gw.rooms["c1"].LoginSlug

	page, err := uc.KidLoginPage(slug)
	if err != nil {
		t.Fatal(err)
	}
	if page.Open || len(page.Kids) != 0 || page.ClassName != "2Б" {
		t.Fatalf("closed page = %+v, want the class name only", page)
	}

	openLesson(t, uc, gw)
	page, _ = uc.KidLoginPage(slug)
	if !page.Open || len(page.Kids) != 2 {
		t.Fatalf("open page = %+v, want two kids", page)
	}

	setKidClock(kidTestStart.Add(91 * time.Minute))
	page, _ = uc.KidLoginPage(slug)
	if page.Open || len(page.Kids) != 0 {
		t.Fatal("the window must close by itself")
	}

	if _, err := uc.KidLoginPage("nope"); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("unknown slug: %v", err)
	}
}

func TestKidLoginWithPictures(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	kids := createTwoKids(t, uc)
	masha := kids[0]

	if _, err := uc.ListKids("10", models.Teacher, "c1"); err != nil {
		t.Fatal(err)
	}
	slug := gw.rooms["c1"].LoginSlug
	if _, err := uc.KidLoginWithPictures(slug, masha.UserID, masha.Pictures); !errors.Is(err, teacherclass.ErrKidLoginClosed) {
		t.Fatalf("closed window: %v", err)
	}

	slug = openLesson(t, uc, gw)
	sess, err := uc.KidLoginWithPictures(slug, masha.UserID, masha.Pictures)
	if err != nil {
		t.Fatalf("right pictures: %v", err)
	}
	if sess.UserID != masha.UserID || sess.ClassID != "c1" {
		t.Fatalf("session = %+v", sess)
	}
	if sess.TTL != 90*time.Minute {
		t.Fatalf("ttl = %v, want the rest of the lesson", sess.TTL)
	}
	if gw.kids[masha.UserID].LastLoginAt == nil {
		t.Fatal("last login not recorded")
	}

	if _, err := uc.KidLoginWithPictures(slug, masha.UserID, wrongSecret(masha.Pictures)); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("one picture off: %v", err)
	}
	if _, err := uc.KidLoginWithPictures(slug, masha.UserID, masha.Pictures[:4]); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("four of five: %v", err)
	}
	if _, err := uc.KidLoginWithPictures(slug, "424242", masha.Pictures); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("unknown child: %v", err)
	}
	if _, err := uc.KidLoginWithPictures(slug, masha.UserID, []int{1}); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("one picture: %v", err)
	}
}

func TestKidLockedAfterFiveWrongPairs(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	kids := createTwoKids(t, uc)
	petya := kids[1]
	slug := openLesson(t, uc, gw)
	wrong := wrongSecret(petya.Pictures)

	for i := 1; i <= 4; i++ {
		if _, err := uc.KidLoginWithPictures(slug, petya.UserID, wrong); !errors.Is(err, teacherclass.ErrBadKidSecret) {
			t.Fatalf("try %d: %v", i, err)
		}
	}
	if _, err := uc.KidLoginWithPictures(slug, petya.UserID, wrong); !errors.Is(err, teacherclass.ErrKidLocked) {
		t.Fatalf("fifth wrong pair: %v", err)
	}
	if _, err := uc.KidLoginWithPictures(slug, petya.UserID, petya.Pictures); !errors.Is(err, teacherclass.ErrKidLocked) {
		t.Fatalf("right pictures while locked: %v", err)
	}
	if err := uc.UnlockKid("10", models.Teacher, "c1", petya.UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.KidLoginWithPictures(slug, petya.UserID, petya.Pictures); err != nil {
		t.Fatalf("after unlock: %v", err)
	}
}

func TestResetPicturesUnlocksAndChangesSecret(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	kids := createTwoKids(t, uc)
	now := time.Now()
	gw.kids[kids[0].UserID].LockedAt = &now

	dto, err := uc.ResetKidPictures("10", models.Teacher, "c1", kids[0].UserID)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Locked || len(dto.Pictures) != models.KidSecretLength {
		t.Fatalf("after reset = %+v", dto)
	}
	if _, err := uc.ResetKidPictures("10", models.Teacher, "c2", kids[0].UserID); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("kid of another class: %v", err)
	}
}

func TestKidLoginWithCard(t *testing.T) {
	uc, gw, lms := newKidUseCase(t)
	kids := createTwoKids(t, uc)

	cards, err := uc.IssueKidCards("10", models.Teacher, "c1", []string{kids[0].UserID, kids[0].UserID})
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 {
		t.Fatalf("cards = %d, want duplicates dropped", len(cards))
	}
	token := cards[0].URL[strings.Index(cards[0].URL, "#")+1:]
	if !strings.Contains(cards[0].URL, "/c/card#") || len(token) < 20 {
		t.Fatalf("card url = %q", cards[0].URL)
	}
	if *gw.kids[kids[0].UserID].CardTokenHash == token {
		t.Fatal("the token is stored as is, want a hash")
	}

	// A card works outside the lesson.
	sess, err := uc.KidLoginWithCard(token)
	if err != nil {
		t.Fatalf("card: %v", err)
	}
	if sess.UserID != kids[0].UserID || sess.TTL != defaultKidSessionTTL {
		t.Fatalf("session = %+v", sess)
	}

	// A new card replaces the old one.
	if _, err := uc.IssueKidCards("10", models.Teacher, "c1", []string{kids[0].UserID}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.KidLoginWithCard(token); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("old card: %v", err)
	}

	// Removed from the class or deactivated in LMS: no sign-in.
	cards, _ = uc.IssueKidCards("10", models.Teacher, "c1", []string{kids[1].UserID})
	token = cards[0].URL[strings.Index(cards[0].URL, "#")+1:]
	lms.inactive[kids[1].UserID] = true
	if _, err := uc.KidLoginWithCard(token); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("inactive: %v", err)
	}
	lms.inactive[kids[1].UserID] = false
	delete(gw.members["c1"], kids[1].UserID)
	if _, err := uc.KidLoginWithCard(token); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("removed from class: %v", err)
	}
	if err := uc.ReturnKid("10", models.Teacher, "c1", kids[1].UserID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.KidLoginWithCard(token); err != nil {
		t.Fatalf("returned: %v", err)
	}
	if _, err := uc.KidLoginWithCard(""); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("empty token: %v", err)
	}
}

func TestKidAccountJoinsNoOtherClass(t *testing.T) {
	uc, _, _ := newKidUseCase(t)
	kids := createTwoKids(t, uc)
	if _, err := uc.JoinClass(kids[0].UserID, "s2"); !errors.Is(err, teacherclass.ErrKidAccount) {
		t.Fatalf("join another class: %v", err)
	}
}

func TestRotateInviteMovesTheClassPage(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	createTwoKids(t, uc)
	before := gw.rooms["c1"].LoginSlug
	uc.rotateKidLoginSlug("c1")
	if after := gw.rooms["c1"].LoginSlug; after == before || len(after) != kidLoginSlugLen {
		t.Fatalf("slug %q -> %q, want a new one", before, after)
	}
}

func TestKidSessionTTL(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	kids := createTwoKids(t, uc)
	start := kidTestStart
	if _, err := uc.OpenKidLogin("10", models.Teacher, "c1", 240); err != nil {
		t.Fatal(err)
	}
	slug := gw.rooms["c1"].LoginSlug
	sess, err := uc.KidLoginWithPictures(slug, kids[0].UserID, kids[0].Pictures)
	if err != nil {
		t.Fatal(err)
	}
	if sess.TTL != defaultKidSessionTTL {
		t.Fatalf("4 h lesson: ttl = %v, want the 3 h cap", sess.TTL)
	}
	setKidClock(start.Add(235 * time.Minute))
	sess, err = uc.KidLoginWithPictures(slug, kids[0].UserID, kids[0].Pictures)
	if err != nil {
		t.Fatal(err)
	}
	if sess.TTL != minKidSessionTTL {
		t.Fatalf("5 min left: ttl = %v, want %v", sess.TTL, minKidSessionTTL)
	}
	if _, err := uc.OpenKidLogin("10", models.Teacher, "c1", 5); !errors.Is(err, teacherclass.ErrBadRequest) {
		t.Fatalf("5 min window: %v", err)
	}
}

func TestKidLoginOffWithoutGateway(t *testing.T) {
	uc := &UseCaseImpl{gw: newLocalGateway()}
	if ok, err := uc.kidElsewhere("1", "c1"); ok || err != nil {
		t.Fatalf("kidElsewhere = %v, %v", ok, err)
	}
	uc.rotateKidLoginSlug("c1")
	if _, err := uc.KidLoginWithCard("x"); !errors.Is(err, teacherclass.ErrBadKidSecret) {
		t.Fatalf("card: %v", err)
	}
}

func TestRoomCodeLeadsToTheClassPageDuringTheLesson(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	createTwoKids(t, uc)

	room, err := uc.OpenKidLogin("10", models.Teacher, "c1", 90)
	if err != nil {
		t.Fatal(err)
	}
	if !isRoomCode(room.RoomCode) || !strings.HasSuffix(room.EntryURL, "/k") {
		t.Fatalf("room = %+v, want six digits and the /k address", room)
	}
	slug, err := uc.KidRoomByCode(room.RoomCode)
	if err != nil || slug != gw.rooms["c1"].LoginSlug {
		t.Fatalf("by code = %q, %v", slug, err)
	}

	// Opening again during the lesson (a longer window) keeps the code on the projector.
	again, err := uc.OpenKidLogin("10", models.Teacher, "c1", 120)
	if err != nil || again.RoomCode != room.RoomCode {
		t.Fatalf("reopened: code %q -> %q, %v", room.RoomCode, again.RoomCode, err)
	}

	for _, bad := range []string{"", "12345", "1234567", "12a456", " "} {
		if _, err := uc.KidRoomByCode(bad); !errors.Is(err, teacherclass.ErrNotFound) {
			t.Fatalf("code %q: %v", bad, err)
		}
	}

	// Closed: the code is gone; the next lesson gets a new one.
	closed, err := uc.CloseKidLogin("10", models.Teacher, "c1")
	if err != nil || closed.RoomCode != "" || gw.rooms["c1"].RoomCode != nil {
		t.Fatalf("closed room = %+v, %v", closed, err)
	}
	if _, err := uc.KidRoomByCode(room.RoomCode); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("code of a closed lesson: %v", err)
	}

	// The window ran out by itself: the code stays in the row but leads nowhere.
	next, _ := uc.OpenKidLogin("10", models.Teacher, "c1", 15)
	setKidClock(kidTestStart.Add(16 * time.Minute))
	if _, err := uc.KidRoomByCode(next.RoomCode); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("code after the window: %v", err)
	}
	reopened, _ := uc.OpenKidLogin("10", models.Teacher, "c1", 90)
	if reopened.RoomCode == "" {
		t.Fatal("a new lesson after an expired one needs a code")
	}
}

func TestRoomCodeTakenByAnotherClassIsReplaced(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	taken := "123456"
	gw.rooms["c2"] = &models.ClassKidLoginRoomDB{ClassID: "c2", LoginSlug: strings.Repeat("b", 16), RoomCode: &taken}
	codes := []string{taken, taken, "654321"}
	prev := genRoomCode
	genRoomCode = func() (string, error) {
		c := codes[0]
		codes = codes[1:]
		return c, nil
	}
	t.Cleanup(func() { genRoomCode = prev })

	room, err := uc.OpenKidLogin("10", models.Teacher, "c1", 90)
	if err != nil || room.RoomCode != "654321" {
		t.Fatalf("room = %+v, %v; want the third code", room, err)
	}
}

func TestRoomCodeOfAnLmsClassLeadsNowhere(t *testing.T) {
	uc, gw, _ := newKidUseCase(t)
	code := "111111"
	until := kidTestStart.Add(time.Hour)
	gw.rooms["lms"] = &models.ClassKidLoginRoomDB{ClassID: "lms", LoginSlug: strings.Repeat("c", 16), OpenUntil: &until, RoomCode: &code}
	if _, err := uc.KidRoomByCode(code); !errors.Is(err, teacherclass.ErrNotFound) {
		t.Fatalf("LMS class: %v", err)
	}
}
