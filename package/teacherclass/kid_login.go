package teacherclass

import (
	"errors"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

// Class sign-in for young students: architecture/ADR_CLASS_KID_LOGIN.md.

var (
	// The class is an Open edX cohort: its students need an LMS session, class sign-in gives none.
	ErrLmsLinkedClass = errors.New("lms_linked_class")
	// The teacher did not confirm that the parents agreed to the processing of the data.
	ErrKidConsentRequired = errors.New("kid_consent_required")
	// The class already has classes.kidLogin.maxPerClass kid accounts.
	ErrTooManyKids = errors.New("too_many_kids")
	// A kid account belongs to its own class and joins no other.
	ErrKidAccount = errors.New("kid_account")
	// Wrong pictures, a child not in this class, or a card that does not work: one answer for all.
	ErrBadKidSecret = errors.New("bad_pictures")
	// The lesson window is closed: picture sign-in waits for the teacher.
	ErrKidLoginClosed = errors.New("kid_login_closed")
	// Too many wrong pictures in a row: the teacher unlocks the child.
	ErrKidLocked = errors.New("kid_locked")
)

// KidDTO is a kid account as its teachers see it.
type KidDTO struct {
	UserID      string     `json:"userId"`
	Name        string     `json:"name"`
	Avatar      int        `json:"avatar"`
	Pictures    []int      `json:"pictures"`
	Locked      bool       `json:"locked"`
	InClass     bool       `json:"inClass"`
	HasCard     bool       `json:"hasCard"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

// KidRoomDTO is the class sign-in page. OpenUntil and RoomCode are set while picture sign-in
// is open: a laptop opens EntryURL (the short /k address) and types RoomCode.
type KidRoomDTO struct {
	LoginSlug string     `json:"loginSlug"`
	LoginURL  string     `json:"loginUrl"`
	EntryURL  string     `json:"entryUrl"`
	OpenUntil *time.Time `json:"openUntil,omitempty"`
	RoomCode  string     `json:"roomCode,omitempty"`
}

type KidRosterDTO struct {
	Room KidRoomDTO `json:"room"`
	Kids []KidDTO   `json:"kids"`
	// MaxKids a class may have; MaxBatch names a teacher adds at once.
	MaxKids  int `json:"maxKids"`
	MaxBatch int `json:"maxBatch"`
}

// KidCardDTO is a printed card. Token and URL are shown once: only a hash is kept.
type KidCardDTO struct {
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	Avatar   int    `json:"avatar"`
	Pictures []int  `json:"pictures"`
	URL      string `json:"url"`
}

// KidLoginPageDTO is the public class page. Kids are listed only while the window is open.
type KidLoginPageDTO struct {
	ClassName string            `json:"className"`
	Open      bool              `json:"open"`
	OpenUntil *time.Time        `json:"openUntil,omitempty"`
	Kids      []KidLoginPageKid `json:"kids"`
}

type KidLoginPageKid struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Avatar int    `json:"avatar"`
}

// KidSession is who signed in and for how long; the handler issues the ЛК session.
type KidSession struct {
	UserID   string
	Username string
	Email    string
	ClassID  string
	// Name and Avatar are shown in the ЛК; LoginSlug is the class page to come back to.
	Name      string
	Avatar    int
	LoginSlug string
	TTL       time.Duration
}

// KidLoginGateway is the storage of kid accounts and class pages (Projects PG).
type KidLoginGateway interface {
	GetKidRoom(classID string) (*models.ClassKidLoginRoomDB, error)
	GetKidRoomBySlug(slug string) (*models.ClassKidLoginRoomDB, error)
	GetKidRoomByCode(code string) (*models.ClassKidLoginRoomDB, error)
	// SaveKidRoom: ErrConflict when another class holds the room code.
	SaveKidRoom(room *models.ClassKidLoginRoomDB) error
	CreateKidLogin(row *models.ClassKidLoginDB) error
	ListKidLogins(classID string) ([]models.ClassKidLoginDB, error)
	CountKidLogins(classID string) (int64, error)
	GetKidLogin(userID string) (*models.ClassKidLoginDB, error)
	GetKidLoginByCardHash(hash string) (*models.ClassKidLoginDB, error)
	SetKidPictures(userID string, pictures []int) error
	SetKidCardHash(userID, hash string) error
	UnlockKid(userID string) error
	// RecordKidLoginFailure counts a wrong secret and locks the child at lockAfter in a row.
	RecordKidLoginFailure(userID string, lockAfter int) (locked bool, err error)
	RecordKidLoginSuccess(userID string, at time.Time) error
}

// KidLoginUseCase: teachers make kid accounts and open the lesson; children sign in.
type KidLoginUseCase interface {
	ListKids(teacherID string, role models.Role, classID string) (*KidRosterDTO, error)
	CreateKids(teacherID string, role models.Role, classID string, names []string, consent bool) ([]KidDTO, error)
	ResetKidPictures(teacherID string, role models.Role, classID, kidID string) (*KidDTO, error)
	UnlockKid(teacherID string, role models.Role, classID, kidID string) error
	ReturnKid(teacherID string, role models.Role, classID, kidID string) error
	IssueKidCards(teacherID string, role models.Role, classID string, kidIDs []string) ([]KidCardDTO, error)
	OpenKidLogin(teacherID string, role models.Role, classID string, minutes int) (*KidRoomDTO, error)
	CloseKidLogin(teacherID string, role models.Role, classID string) (*KidRoomDTO, error)

	// KidRoomByCode: the class page slug behind a room code of an open lesson.
	KidRoomByCode(code string) (string, error)
	KidLoginPage(slug string) (*KidLoginPageDTO, error)
	KidLoginWithPictures(slug, kidID string, pictures []int) (*KidSession, error)
	KidLoginWithCard(token string) (*KidSession, error)
}
