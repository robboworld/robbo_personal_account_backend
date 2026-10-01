//go:build external

// Calls the live edx test API; run with: go test -tags external ./package/edx/...
package users

import (
	"github.com/go-playground/assert/v2"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/config"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx/usecase"
	"log"
	"testing"
)

func TestGetUser(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edxUseCase := usecase.SetupEdxApiUseCase()

	expect := []byte("{\"username\":\"edxsom\"}")
	correct, _ := edxUseCase.UseCase.GetUser()
	assert.Equal(t, expect, correct)

}

func TestEdxApiUseCaseImpl_Login(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edxUseCase := usecase.SetupEdxApiUseCase()
	testTable := []struct {
		name          string
		email         string
		password      string
		expectedError error
	}{
		{
			name:          "Ok",
			email:         "edxsom@test.com",
			password:      "123456",
			expectedError: nil,
		},

		{
			name:          "Email or password incorrect",
			email:         "dsadddas",
			password:      "dsadad",
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name:          "Email or password is empty",
			email:         "",
			password:      "",
			expectedError: edx.ErrIncorrectInputParam,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			expect := testCase.expectedError
			_, correct := edxUseCase.UseCase.Login(testCase.email, testCase.password)

			assert.Equal(t, expect, correct)
		})
	}
}

func TestEdxApiUseCaseImpl_PostRegistration(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edxUseCase := usecase.SetupEdxApiUseCase()
	testTable := []struct {
		name                string
		registrationMessage edx.RegistrationForm
		expectedError       error
	}{
		{
			name: "Ok",
			registrationMessage: edx.RegistrationForm{
				Email:          "insomnia_testrrwds323fsd22dasf3@fake.email",
				Username:       "InsomniaTedasd122fsdfda3",
				Name:           "SomeTestNafdsme12ddsds3",
				Password:       "123456",
				TermsOfService: "true",
			},
			expectedError: nil,
		},

		{
			name: "Password is empty",
			registrationMessage: edx.RegistrationForm{
				Email:          "insomnia_testrrw223@fake.email",
				Username:       "InsomniaTest31223",
				Name:           "SomeTestName123",
				Password:       "",
				TermsOfService: "true",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name: "Email is empty",
			registrationMessage: edx.RegistrationForm{
				Email:          "",
				Username:       "InsomniaTest31223",
				Name:           "SomeTestName123",
				Password:       "123456",
				TermsOfService: "true",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name: "Username is empty",
			registrationMessage: edx.RegistrationForm{
				Email:          "nsomnia_testrrw223@fake.email",
				Username:       "",
				Name:           "SomeTestName123",
				Password:       "123456",
				TermsOfService: "true",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name: "Name is empty",
			registrationMessage: edx.RegistrationForm{
				Email:          "nsomnia_testrrw223@fake.email",
				Username:       "dsadasd",
				Name:           "",
				Password:       "123456",
				TermsOfService: "true",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name: "TermsOfService is empty",
			registrationMessage: edx.RegistrationForm{
				Email:          "nsomnia_testrrw223@fake.email",
				Username:       "dsadasd",
				Name:           "gdgsdfsfs",
				Password:       "123456",
				TermsOfService: "",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},

		{
			name: "All params is empty",
			registrationMessage: edx.RegistrationForm{
				Email:          "",
				Username:       "",
				Name:           "",
				Password:       "",
				TermsOfService: "",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedError
			_, correct := edxUseCase.UseCase.PostRegistration(testCase.registrationMessage)
			assert.Equal(t, expect, correct)
		})
	}

}
