package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/teacherclass"
	"github.com/spf13/viper"
)

// Class sign-in for young students: architecture/ADR_CLASS_KID_LOGIN.md. A teacher types the
// names, the ЛК creates their LMS users, and a child signs in on the class page with their
// avatar and five pictures in order (while the teacher keeps the lesson open) or with a printed
// QR card.

const (
	defaultKidMaxBatch      = 40
	defaultKidMaxPerClass   = 60
	defaultKidLockAfter     = 5
	defaultKidSessionTTL    = 3 * time.Hour
	minKidSessionTTL        = 30 * time.Minute
	minKidLoginWindow       = 15
	maxKidLoginWindow       = 240
	defaultKidLoginWindow   = 90
	maxKidNameRunes         = 40
	kidEmailDomain          = "kids.robbo.invalid"
	kidUsernamePrefix       = "k-"
	kidLoginSlugLen         = 16
	kidCardTokenBytes       = 16
	kidRandomUsernameLength = 10
)

func init() {
	historyCategories["kids"] = []string{
		models.HistoryKidsCreated, models.HistoryKidLoginOpened, models.HistoryKidLoginClosed,
		models.HistoryKidPicturesSet, models.HistoryKidCardsIssued, models.HistoryKidUnlocked,
		models.HistoryKidReturned, models.HistoryKidLogin,
	}
}

// kidGateway: the class gateway stores kid accounts too (Projects PG).
func kidGateway(gw teacherclass.Gateway) teacherclass.KidLoginGateway {
	kids, _ := gw.(teacherclass.KidLoginGateway)
	return kids
}

func kidConfigInt(key string, def int) int {
	if v := viper.GetInt(key); v > 0 {
		return v
	}
	return def
}

func kidMaxBatch() int    { return kidConfigInt("classes.kidLogin.maxBatch", defaultKidMaxBatch) }
func kidMaxPerClass() int { return kidConfigInt("classes.kidLogin.maxPerClass", defaultKidMaxPerClass) }
func kidLockAfter() int   { return kidConfigInt("classes.kidLogin.lockAfter", defaultKidLockAfter) }

func kidSessionTTL() time.Duration {
	if d := viper.GetDuration("classes.kidLogin.sessionTTL"); d > 0 {
		return d
	}
	return defaultKidSessionTTL
}

// kidNow is the clock; tests move it.
var kidNow = func() time.Time { return time.Now().UTC() }

// createKidLMSUser makes the LMS user of a kid account. The password cannot be used ("!"
// prefix, as Django's unusable password): the child signs in only through the class page.
var createKidLMSUser = func(username, email, name string) (int64, error) {
	writer, err := lmsdb.NewWriterFromConfig()
	if err != nil {
		return 0, err
	}
	defer writer.Close()
	unusable, err := randomToken(30)
	if err != nil {
		return 0, err
	}
	return writer.CreateUserWithProfile(username, email, "!"+unusable, name, "", false)
}

// kidLMSUser is what the session needs from LMS: the login and whether the account is active.
type kidLMSUser struct {
	Username string
	Email    string
	Active   bool
}

var lookupKidLMSUser = func(userID string) (*kidLMSUser, error) {
	id, err := lmsdb.ParseLMSUserID(userID)
	if err != nil {
		return nil, err
	}
	reader, err := lmsdb.NewReaderFromConfig()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	row, err := reader.LookupAuthUserByID(id)
	if err != nil || row == nil {
		return nil, err
	}
	login, err := reader.LookupAuthUserForLogin(row.Username)
	if err != nil || login == nil || login.ID != id {
		return nil, err
	}
	return &kidLMSUser{Username: login.Username, Email: login.Email, Active: login.IsActive}, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomBase32(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
	return s[:n], nil
}

func randomInt(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

// randomPictures: models.KidSecretLength pictures in order. Repeats are allowed (the teacher's
// choice): 9^5 secrets instead of 9*8*7*6*5.
func randomPictures() ([]int, error) {
	out := make([]int, models.KidSecretLength)
	for i := range out {
		p, err := randomInt(models.KidPictureCount)
		if err != nil {
			return nil, err
		}
		out[i] = p
	}
	return out, nil
}

func picturesArray(pictures []int) []int64 {
	out := make([]int64, len(pictures))
	for i, p := range pictures {
		out[i] = int64(p)
	}
	return out
}

func hashCardToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// cleanKidName keeps what the teacher typed, without control characters and extra spaces.
func cleanKidName(raw string) (string, bool) {
	name := strings.Join(strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}), " ")
	n := utf8.RuneCountInString(name)
	return name, n > 0 && n <= maxKidNameRunes
}

func kidFrontendBase() string {
	base := strings.TrimRight(viper.GetString("frontend.publicUrl"), "/")
	if base == "" {
		base = "http://localhost:3030"
	}
	return base
}

func kidRoomDTO(room *models.ClassKidLoginRoomDB) teacherclass.KidRoomDTO {
	dto := teacherclass.KidRoomDTO{
		LoginSlug: room.LoginSlug,
		LoginURL:  kidFrontendBase() + "/c/" + room.LoginSlug,
		EntryURL:  kidFrontendBase() + "/k",
	}
	if room.IsOpen(kidNow()) {
		t := *room.OpenUntil
		dto.OpenUntil = &t
		if room.RoomCode != nil {
			dto.RoomCode = *room.RoomCode
		}
	}
	return dto
}

func picturesOf(row *models.ClassKidLoginDB) []int {
	out := make([]int, len(row.Pictures))
	for i, p := range row.Pictures {
		out[i] = int(p)
	}
	return out
}

func kidDTO(row *models.ClassKidLoginDB, inClass bool) teacherclass.KidDTO {
	return teacherclass.KidDTO{
		UserID:      row.UserID,
		Name:        row.DisplayName,
		Avatar:      row.Avatar,
		Pictures:    picturesOf(row),
		Locked:      row.LockedAt != nil,
		InClass:     inClass,
		HasCard:     row.CardTokenHash != nil,
		CreatedAt:   row.CreatedAt,
		LastLoginAt: row.LastLoginAt,
	}
}

// requireKidClass: a teacher of a class kept in the ЛК.
func (u *UseCaseImpl) requireKidClass(teacherID string, role models.Role, classID string) (*models.ClassInviteDB, error) {
	if u.kids == nil {
		return nil, teacherclass.ErrNotFound
	}
	inv, err := u.requireTeaching(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	if inv.LmsLinked {
		return nil, teacherclass.ErrLmsLinkedClass
	}
	return inv, nil
}

// kidRoom is the class page; the first call makes it.
func (u *UseCaseImpl) kidRoom(classID string) (*models.ClassKidLoginRoomDB, error) {
	room, err := u.kids.GetKidRoom(classID)
	if err == nil {
		return room, nil
	}
	if !errors.Is(err, teacherclass.ErrNotFound) {
		return nil, err
	}
	slug, err := randomBase32(kidLoginSlugLen)
	if err != nil {
		return nil, err
	}
	room = &models.ClassKidLoginRoomDB{ClassID: classID, LoginSlug: slug}
	if err := u.kids.SaveKidRoom(room); err != nil {
		return nil, err
	}
	return room, nil
}

// rotateKidLoginSlug moves the class page with the invite: the old link stops listing names.
func (u *UseCaseImpl) rotateKidLoginSlug(classID string) {
	if u.kids == nil {
		return
	}
	room, err := u.kids.GetKidRoom(classID)
	if err != nil {
		if !errors.Is(err, teacherclass.ErrNotFound) {
			log.Printf("teacherclass: class %s kid room: %v", classID, err)
		}
		return
	}
	slug, err := randomBase32(kidLoginSlugLen)
	if err == nil {
		room.LoginSlug = slug
		err = u.kids.SaveKidRoom(room)
	}
	if err != nil {
		log.Printf("teacherclass: class %s rotate kid login slug: %v", classID, err)
	}
}

// kidElsewhere: userID is a kid account of another class. Kid accounts join no other class.
func (u *UseCaseImpl) kidElsewhere(userID, classID string) (bool, error) {
	if u.kids == nil {
		return false, nil
	}
	row, err := u.kids.GetKidLogin(userID)
	if errors.Is(err, teacherclass.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.ClassID != classID, nil
}

func (u *UseCaseImpl) memberSet(classID string) (map[string]bool, error) {
	rows, err := u.gw.ListMembers(classID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.UserID] = true
	}
	return out, nil
}

func (u *UseCaseImpl) ListKids(teacherID string, role models.Role, classID string) (*teacherclass.KidRosterDTO, error) {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	room, err := u.kidRoom(inv.ID)
	if err != nil {
		return nil, err
	}
	rows, err := u.kids.ListKidLogins(inv.ID)
	if err != nil {
		return nil, err
	}
	members, err := u.memberSet(inv.ID)
	if err != nil {
		return nil, err
	}
	out := &teacherclass.KidRosterDTO{
		Room:     kidRoomDTO(room),
		Kids:     make([]teacherclass.KidDTO, 0, len(rows)),
		MaxKids:  kidMaxPerClass(),
		MaxBatch: kidMaxBatch(),
	}
	for i := range rows {
		out.Kids = append(out.Kids, kidDTO(&rows[i], members[rows[i].UserID]))
	}
	return out, nil
}

// pickAvatar: an animal nobody in the class has; when all are taken, one of the least used.
func pickAvatar(used map[int]int) (int, error) {
	least := -1
	var pool []int
	for a := 0; a < models.KidAvatarCount; a++ {
		n := used[a]
		switch {
		case least == -1 || n < least:
			least = n
			pool = []int{a}
		case n == least:
			pool = append(pool, a)
		}
	}
	i, err := randomInt(len(pool))
	if err != nil {
		return 0, err
	}
	return pool[i], nil
}

func (u *UseCaseImpl) CreateKids(teacherID string, role models.Role, classID string, names []string, consent bool) ([]teacherclass.KidDTO, error) {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	if !consent {
		return nil, teacherclass.ErrKidConsentRequired
	}
	clean := make([]string, 0, len(names))
	for _, raw := range names {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		name, ok := cleanKidName(raw)
		if !ok {
			return nil, teacherclass.ErrBadRequest
		}
		clean = append(clean, name)
	}
	if len(clean) == 0 || len(clean) > kidMaxBatch() {
		return nil, teacherclass.ErrBadRequest
	}
	if _, err := u.kidRoom(inv.ID); err != nil {
		return nil, err
	}
	existing, err := u.kids.ListKidLogins(inv.ID)
	if err != nil {
		return nil, err
	}
	if len(existing)+len(clean) > kidMaxPerClass() {
		return nil, teacherclass.ErrTooManyKids
	}
	used := make(map[int]int, len(existing))
	for _, row := range existing {
		used[row.Avatar]++
	}

	out := make([]teacherclass.KidDTO, 0, len(clean))
	for _, name := range clean {
		row, err := u.createKid(inv, teacherID, name, used)
		if err != nil {
			// The ones made so far stay: the teacher sees them and adds the rest again.
			log.Printf("teacherclass: class %s create kid account: %v", inv.ID, err)
			if len(out) == 0 {
				return nil, err
			}
			break
		}
		used[row.Avatar]++
		out = append(out, kidDTO(row, true))
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryKidsCreated, models.ClassHistoryEventDB{
		Title:  "consent",
		Detail: strconv.Itoa(len(out)),
	})
	return out, nil
}

func (u *UseCaseImpl) createKid(inv *models.ClassInviteDB, teacherID, name string, used map[int]int) (*models.ClassKidLoginDB, error) {
	avatar, err := pickAvatar(used)
	if err != nil {
		return nil, err
	}
	pictures, err := randomPictures()
	if err != nil {
		return nil, err
	}
	var lmsID int64
	for attempt := 0; ; attempt++ {
		suffix, err := randomBase32(kidRandomUsernameLength)
		if err != nil {
			return nil, err
		}
		username := kidUsernamePrefix + suffix
		lmsID, err = createKidLMSUser(username, username+"@"+kidEmailDomain, name)
		if err == nil {
			break
		}
		if attempt >= 2 {
			return nil, fmt.Errorf("lms user: %w", err)
		}
	}
	row := &models.ClassKidLoginDB{
		UserID:      strconv.FormatInt(lmsID, 10),
		ClassID:     inv.ID,
		DisplayName: name,
		Avatar:      avatar,
		Pictures:    picturesArray(pictures),
		CreatedBy:   teacherID,
		CreatedAt:   kidNow(),
	}
	if err := u.kids.CreateKidLogin(row); err != nil {
		return nil, fmt.Errorf("kid login row for LMS user %d: %w", lmsID, err)
	}
	if err := u.addKidToClass(inv, row.UserID); err != nil {
		return nil, fmt.Errorf("add kid %s to class: %w", row.UserID, err)
	}
	return row, nil
}

// addKidToClass puts the child on the roster with copies of the open assignments, like a late
// joiner, but without the "joined" notice: the teacher added them.
func (u *UseCaseImpl) addKidToClass(inv *models.ClassInviteDB, userID string) error {
	if err := u.gw.AddMember(inv.ID, userID); err != nil {
		return err
	}
	assignments, err := u.gw.ListAssignments(inv.EdxCourseID, inv.EdxCohortID)
	if err != nil {
		log.Printf("teacherclass: class %s assignments for kid %s: %v", inv.ID, userID, err)
		return nil
	}
	for i := range assignments {
		a := &assignments[i]
		// A file submission starts with the student's first file: nothing to issue.
		if a.ClosedAt != nil || a.IsFileAssignment() {
			continue
		}
		if _, err := u.gw.FindStudentAssignmentProject(a.ID, userID); err == nil {
			continue
		}
		tpl, err := u.assignmentTemplate(a)
		if err != nil {
			continue
		}
		if _, err := u.gw.ForkProjectForStudent(tpl, userID, a.Title, a); err != nil {
			log.Printf("teacherclass: class %s fork %s for kid %s: %v", inv.ID, a.ID, userID, err)
		}
	}
	return nil
}

// classKid: the kid account of this class.
func (u *UseCaseImpl) classKid(classID, kidID string) (*models.ClassKidLoginDB, error) {
	row, err := u.kids.GetKidLogin(strings.TrimSpace(kidID))
	if err != nil {
		return nil, err
	}
	if row.ClassID != classID {
		return nil, teacherclass.ErrNotFound
	}
	return row, nil
}

func (u *UseCaseImpl) ResetKidPictures(teacherID string, role models.Role, classID, kidID string) (*teacherclass.KidDTO, error) {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	row, err := u.classKid(inv.ID, kidID)
	if err != nil {
		return nil, err
	}
	pictures, err := randomPictures()
	if err != nil {
		return nil, err
	}
	if err := u.kids.SetKidPictures(row.UserID, pictures); err != nil {
		return nil, err
	}
	row.Pictures = picturesArray(pictures)
	row.LockedAt = nil
	row.FailedAttempts = 0
	u.recordTeacherAction(inv, teacherID, role, models.HistoryKidPicturesSet, models.ClassHistoryEventDB{TargetUserID: row.UserID})
	inClass, err := u.gw.IsMemberOfAny(row.UserID, []string{inv.ID})
	if err != nil {
		return nil, err
	}
	dto := kidDTO(row, inClass)
	return &dto, nil
}

func (u *UseCaseImpl) UnlockKid(teacherID string, role models.Role, classID, kidID string) error {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return err
	}
	row, err := u.classKid(inv.ID, kidID)
	if err != nil {
		return err
	}
	if err := u.kids.UnlockKid(row.UserID); err != nil {
		return err
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryKidUnlocked, models.ClassHistoryEventDB{TargetUserID: row.UserID})
	return nil
}

// ReturnKid puts a kid account that left or was removed back on the roster.
func (u *UseCaseImpl) ReturnKid(teacherID string, role models.Role, classID, kidID string) error {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return err
	}
	row, err := u.classKid(inv.ID, kidID)
	if err != nil {
		return err
	}
	if in, err := u.gw.IsMemberOfAny(row.UserID, []string{inv.ID}); err != nil {
		return err
	} else if in {
		return nil
	}
	if err := u.addKidToClass(inv, row.UserID); err != nil {
		return err
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryKidReturned, models.ClassHistoryEventDB{TargetUserID: row.UserID})
	return nil
}

// IssueKidCards makes new card tokens: the cards printed before stop working.
func (u *UseCaseImpl) IssueKidCards(teacherID string, role models.Role, classID string, kidIDs []string) ([]teacherclass.KidCardDTO, error) {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	if len(kidIDs) == 0 || len(kidIDs) > kidMaxPerClass() {
		return nil, teacherclass.ErrBadRequest
	}
	seen := make(map[string]bool, len(kidIDs))
	out := make([]teacherclass.KidCardDTO, 0, len(kidIDs))
	for _, id := range kidIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		row, err := u.classKid(inv.ID, id)
		if err != nil {
			return nil, err
		}
		token, err := randomToken(kidCardTokenBytes)
		if err != nil {
			return nil, err
		}
		if err := u.kids.SetKidCardHash(row.UserID, hashCardToken(token)); err != nil {
			return nil, err
		}
		out = append(out, teacherclass.KidCardDTO{
			UserID:   row.UserID,
			Name:     row.DisplayName,
			Avatar:   row.Avatar,
			Pictures: picturesOf(row),
			URL:      kidFrontendBase() + "/c/card#" + token,
		})
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryKidCardsIssued, models.ClassHistoryEventDB{Detail: strconv.Itoa(len(out))})
	return out, nil
}

func (u *UseCaseImpl) OpenKidLogin(teacherID string, role models.Role, classID string, minutes int) (*teacherclass.KidRoomDTO, error) {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	if minutes == 0 {
		minutes = defaultKidLoginWindow
	}
	if minutes < minKidLoginWindow || minutes > maxKidLoginWindow {
		return nil, teacherclass.ErrBadRequest
	}
	room, err := u.kidRoom(inv.ID)
	if err != nil {
		return nil, err
	}
	// A lesson that is still open keeps its code: it is on the projector already.
	keepCode := room.IsOpen(kidNow()) && room.RoomCode != nil
	until := kidNow().Add(time.Duration(minutes) * time.Minute)
	room.OpenUntil = &until
	if err := u.saveRoomWithCode(room, keepCode); err != nil {
		return nil, err
	}
	u.recordTeacherAction(inv, teacherID, role, models.HistoryKidLoginOpened, models.ClassHistoryEventDB{Detail: until.Format(time.RFC3339)})
	dto := kidRoomDTO(room)
	return &dto, nil
}

func (u *UseCaseImpl) CloseKidLogin(teacherID string, role models.Role, classID string) (*teacherclass.KidRoomDTO, error) {
	inv, err := u.requireKidClass(teacherID, role, classID)
	if err != nil {
		return nil, err
	}
	room, err := u.kidRoom(inv.ID)
	if err != nil {
		return nil, err
	}
	wasOpen := room.IsOpen(kidNow())
	room.OpenUntil = nil
	room.RoomCode = nil
	if err := u.kids.SaveKidRoom(room); err != nil {
		return nil, err
	}
	if wasOpen {
		u.recordTeacherAction(inv, teacherID, role, models.HistoryKidLoginClosed, models.ClassHistoryEventDB{})
	}
	dto := kidRoomDTO(room)
	return &dto, nil
}

// genRoomCode: six random digits; tests replace it to force a taken code.
var genRoomCode = func() (string, error) {
	n, err := randomInt(1000000)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n), nil
}

const roomCodeAttempts = 5

// saveRoomWithCode saves the room with a new code unless keepCode, picking another code when
// a class already holds this one (codes left on expired windows count too: the index is unique).
func (u *UseCaseImpl) saveRoomWithCode(room *models.ClassKidLoginRoomDB, keepCode bool) error {
	if keepCode {
		return u.kids.SaveKidRoom(room)
	}
	for attempt := 0; ; attempt++ {
		code, err := genRoomCode()
		if err != nil {
			return err
		}
		room.RoomCode = &code
		err = u.kids.SaveKidRoom(room)
		if !errors.Is(err, teacherclass.ErrConflict) || attempt+1 >= roomCodeAttempts {
			return err
		}
	}
}

func isRoomCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// KidRoomByCode: the page of an open lesson by the code on the projector. A closed lesson's
// code, a wrong code and a class that left class sign-in all answer ErrNotFound.
func (u *UseCaseImpl) KidRoomByCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if u.kids == nil || !isRoomCode(code) {
		return "", teacherclass.ErrNotFound
	}
	room, err := u.kids.GetKidRoomByCode(code)
	if err != nil {
		return "", err
	}
	if !room.IsOpen(kidNow()) {
		return "", teacherclass.ErrNotFound
	}
	if _, _, err := u.kidRoomClass(room.LoginSlug); err != nil {
		return "", err
	}
	return room.LoginSlug, nil
}

// kidRoomClass: the class behind a page slug, if it still takes class sign-in.
func (u *UseCaseImpl) kidRoomClass(slug string) (*models.ClassKidLoginRoomDB, *models.ClassInviteDB, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if u.kids == nil || len(slug) != kidLoginSlugLen {
		return nil, nil, teacherclass.ErrNotFound
	}
	room, err := u.kids.GetKidRoomBySlug(slug)
	if err != nil {
		return nil, nil, err
	}
	inv, err := u.gw.GetInviteByID(room.ClassID) // archived classes are not found
	if err != nil {
		return nil, nil, teacherclass.ErrNotFound
	}
	if inv.LmsLinked {
		return nil, nil, teacherclass.ErrNotFound
	}
	return room, inv, nil
}

func (u *UseCaseImpl) KidLoginPage(slug string) (*teacherclass.KidLoginPageDTO, error) {
	room, inv, err := u.kidRoomClass(slug)
	if err != nil {
		return nil, err
	}
	out := &teacherclass.KidLoginPageDTO{ClassName: inv.DisplayName, Kids: []teacherclass.KidLoginPageKid{}}
	if !room.IsOpen(kidNow()) {
		return out, nil
	}
	out.Open = true
	t := *room.OpenUntil
	out.OpenUntil = &t
	rows, err := u.kids.ListKidLogins(inv.ID)
	if err != nil {
		return nil, err
	}
	members, err := u.memberSet(inv.ID)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if !members[row.UserID] {
			continue
		}
		out.Kids = append(out.Kids, teacherclass.KidLoginPageKid{UserID: row.UserID, Name: row.DisplayName, Avatar: row.Avatar})
	}
	return out, nil
}

func samePictures(row *models.ClassKidLoginDB, pictures []int) bool {
	if len(pictures) != len(row.Pictures) || len(pictures) != models.KidSecretLength {
		return false
	}
	// All compared every time: no early exit to time.
	ok := true
	for i := range pictures {
		ok = ok && int64(pictures[i]) == row.Pictures[i]
	}
	return ok
}

func (u *UseCaseImpl) KidLoginWithPictures(slug, kidID string, pictures []int) (*teacherclass.KidSession, error) {
	room, inv, err := u.kidRoomClass(slug)
	if err != nil {
		return nil, err
	}
	now := kidNow()
	if !room.IsOpen(now) {
		return nil, teacherclass.ErrKidLoginClosed
	}
	row, err := u.classKid(inv.ID, kidID)
	if errors.Is(err, teacherclass.ErrNotFound) {
		return nil, teacherclass.ErrBadKidSecret
	}
	if err != nil {
		return nil, err
	}
	if row.LockedAt != nil {
		return nil, teacherclass.ErrKidLocked
	}
	if !samePictures(row, pictures) {
		locked, err := u.kids.RecordKidLoginFailure(row.UserID, kidLockAfter())
		if err != nil {
			return nil, err
		}
		if locked {
			return nil, teacherclass.ErrKidLocked
		}
		return nil, teacherclass.ErrBadKidSecret
	}
	// Till the end of the lesson, but not cut short right after sign-in, and never long:
	// school computers are shared.
	ttl := room.OpenUntil.Sub(now)
	if ttl < minKidSessionTTL {
		ttl = minKidSessionTTL
	}
	if max := kidSessionTTL(); ttl > max {
		ttl = max
	}
	return u.kidSession(inv, row, ttl, "")
}

func (u *UseCaseImpl) KidLoginWithCard(token string) (*teacherclass.KidSession, error) {
	token = strings.TrimSpace(token)
	if u.kids == nil || token == "" || len(token) > 64 {
		return nil, teacherclass.ErrBadKidSecret
	}
	row, err := u.kids.GetKidLoginByCardHash(hashCardToken(token))
	if errors.Is(err, teacherclass.ErrNotFound) {
		return nil, teacherclass.ErrBadKidSecret
	}
	if err != nil {
		return nil, err
	}
	inv, err := u.gw.GetInviteByID(row.ClassID)
	if err != nil || inv.LmsLinked {
		return nil, teacherclass.ErrBadKidSecret
	}
	return u.kidSession(inv, row, kidSessionTTL(), models.HistoryKidLoginCardTag)
}

// kidSession: the child is still in the class and their LMS account is active.
func (u *UseCaseImpl) kidSession(inv *models.ClassInviteDB, row *models.ClassKidLoginDB, ttl time.Duration, via string) (*teacherclass.KidSession, error) {
	in, err := u.gw.IsMemberOfAny(row.UserID, []string{inv.ID})
	if err != nil {
		return nil, err
	}
	if !in {
		return nil, teacherclass.ErrBadKidSecret
	}
	lms, err := lookupKidLMSUser(row.UserID)
	if err != nil {
		return nil, err
	}
	if lms == nil || !lms.Active {
		return nil, teacherclass.ErrBadKidSecret
	}
	if err := u.kids.RecordKidLoginSuccess(row.UserID, kidNow()); err != nil {
		log.Printf("teacherclass: kid %s last login: %v", row.UserID, err)
	}
	u.recordStudentAction(inv, row.UserID, models.HistoryKidLogin, models.ClassHistoryEventDB{Detail: via})
	sess := &teacherclass.KidSession{
		UserID:   row.UserID,
		Username: lms.Username,
		Email:    lms.Email,
		ClassID:  inv.ID,
		Name:     row.DisplayName,
		Avatar:   row.Avatar,
		TTL:      ttl,
	}
	if room, err := u.kids.GetKidRoom(inv.ID); err == nil {
		sess.LoginSlug = room.LoginSlug
	}
	return sess, nil
}
