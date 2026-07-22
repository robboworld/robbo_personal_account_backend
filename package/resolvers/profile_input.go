package resolvers

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

func userHTTPFromProfileInput(input models.UpdateProfileInput) *models.UserHTTP {
	return &models.UserHTTP{
		ID:               input.ID,
		Email:            input.Email,
		Nickname:         input.Nickname,
		FullName:         input.FullName,
		Firstname:        input.Firstname,
		Lastname:         input.Lastname,
		Middlename:       input.Middlename,
		Bio:              input.Bio,
		LevelOfEducation: input.LevelOfEducation,
		Country:          input.Country,
		YearOfBirth:      input.YearOfBirth,
		Gender:           input.Gender,
		Language:         input.Language,
	}
}
