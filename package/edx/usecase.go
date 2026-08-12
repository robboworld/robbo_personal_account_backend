package edx

type RegistrationForm struct {
	Email          string
	Username       string
	Name           string
	Password       string
	TermsOfService string
}

type NewToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type UseCase interface {
	RefreshToken() (err error)
	GetWithAuth(url string) (resBody []byte, err error)
	PostWithAuth(url string, params map[string]interface{}) (respBody []byte, err error)

	CreateCohort(courseId string, cohortParams map[string]interface{}) (respBody []byte, err error)
	AddStudent(username, courseId string, cohortId int) (respBody []byte, err error)
	GetCohortSettings(courseId string) (respBody []byte, err error)
	SetCohortSettings(courseId string, params map[string]interface{}) (respBody []byte, err error)
	ListCohorts(courseId string) (respBody []byte, err error)
	GetCohort(courseId string, cohortId int) (respBody []byte, err error)
	PatchCohort(courseId string, cohortId int, params map[string]interface{}) (respBody []byte, err error)
	ListCohortUsers(courseId string, cohortId int) (respBody []byte, err error)
	RemoveCohortUser(username, courseId string, cohortId int) (err error)

	GetCoursesByUser() (respBody []byte, err error)
	GetAllPublicCourses(pageNumber int) (respBody []byte, err error)
	GetEnrollments(username string) (respBody []byte, err error)
	GetCourseContent(courseId string) (respBody []byte, err error)
	PostEnrollment(message map[string]interface{}) (respBody []byte, err error)

	GetUser() (respBody []byte, err error)
	PostRegistration(postMessage RegistrationForm) (respBody []byte, err error)
	Login(email, password string) (respBody []byte, err error)
}
