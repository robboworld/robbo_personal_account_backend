package models

// UserHTTP is bound to the GraphQL UserHttp type in gqlgen.yml (not generated) so it can
// carry the write-only Password field that the schema does not expose.
type UserHTTP struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	// Password is input-only (REST create/sign-up bodies); GraphQL UserHttp has no such field
	// and FromCore always blanks it.
	Password         string  `json:"password,omitempty"`
	Role             int     `json:"role"`
	Nickname         string  `json:"nickname"`
	FullName         string  `json:"fullName"`
	Firstname        string  `json:"firstname"`
	Lastname         string  `json:"lastname"`
	Middlename       string  `json:"middlename"`
	Company          string  `json:"company,omitempty"`
	Bio              *string `json:"bio,omitempty"`
	LevelOfEducation *string `json:"levelOfEducation,omitempty"`
	Country          *string `json:"country,omitempty"`
	YearOfBirth      *int    `json:"yearOfBirth,omitempty"`
	Gender           *string `json:"gender,omitempty"`
	Language         *string `json:"language,omitempty"`
	AvatarID         *string `json:"avatarId,omitempty"`
	CreatedAt        string  `json:"createdAt"`
}
