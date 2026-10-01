//go:build external

// Calls the live edx test API; run with: go test -tags external ./package/edx/...
package usecase

import (
	"github.com/go-playground/assert/v2"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/config"
	edxpackage "github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"log"
	"testing"
)

func TestGetUser2(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edx := SetupEdxApiUseCase()

	expect := []byte("{\"username\":\"edxsom\"}")
	correct, _ := edx.UseCase.GetUser()
	assert.Equal(t, expect, correct)

}

func TestEdxApiUseCaseImpl_GetCourseContent2(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edx := SetupEdxApiUseCase()
	testTable := []struct {
		name          string
		courseId      string
		expectedError error
	}{
		{
			name:          "Ok",
			courseId:      "course-v1:TestOrg+02+2022",
			expectedError: nil,
		},

		{
			name:          "Bad courseId",
			courseId:      "Ddssadad",
			expectedError: edxpackage.ErrIncorrectInputParam,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			expect := testCase.expectedError
			_, correct := edx.UseCase.GetCourseContent(testCase.courseId)
			assert.Equal(t, expect, correct)
		})
	}
}

func TestEdxApiUseCaseImpl_GetEnrollments2(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edx := SetupEdxApiUseCase()
	testTable := []struct {
		name          string
		username      string
		expectedError error
	}{
		{
			name:          "Ok",
			username:      "edxsom",
			expectedError: nil,
		},
		{
			name:          "Bad username",
			username:      ".d#42sad",
			expectedError: edxpackage.ErrIncorrectInputParam,
		},
		//{
		//	name:          "Empty username",
		//	username:      "",
		//	expectedError: edxpackage.ErrIncorrectInputParam,
		//},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedError

			_, correct := edx.UseCase.GetEnrollments(testCase.username)
			assert.Equal(t, expect, correct)
		})
	}
}

func TestEdxApiUseCaseImpl_GetAllPublicCourses2(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edx := SetupEdxApiUseCase()
	testTable := []struct {
		name          string
		pageNumber    int
		expectedError error
	}{
		{
			name:          "Ok",
			pageNumber:    1,
			expectedError: nil,
		},
		{
			name:          "Page number is 0",
			pageNumber:    0,
			expectedError: edxpackage.ErrOnReq,
		},
		{
			name:          "Page number more then page count",
			pageNumber:    423423,
			expectedError: edxpackage.ErrOnReq,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedError

			_, correct := edx.UseCase.GetAllPublicCourses(testCase.pageNumber)
			assert.Equal(t, expect, correct)
		})
	}
}

func TestEdxApiUseCaseImpl_PostEnrollment2(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edx := SetupEdxApiUseCase()
	testTable := []struct {
		name          string
		message       map[string]interface{}
		expectedError error
	}{
		{
			name: "Ok",
			message: map[string]interface{}{
				"course_details": map[string]string{
					"course_id": "course-v1:Test_org+01+2022",
				},
				"user": "edxsom",
			},
			expectedError: nil,
		},

		{
			name: "Course id incorrect",
			message: map[string]interface{}{
				"course_details": map[string]string{
					"course_id": "dasda",
				},
				"user": "edxsom",
			},
			expectedError: edxpackage.ErrIncorrectInputParam,
		},
		{
			name: "Username incorrect",
			message: map[string]interface{}{
				"course_details": map[string]string{
					"course_id": "course-v1:Test_org+01+2022",
				},
				"user": "edm",
			},
			expectedError: edxpackage.ErrIncorrectInputParam,
		},
		{
			name: "Empty field courseId",
			message: map[string]interface{}{
				"course_details": map[string]string{
					"course_id": "",
				},
				"user": "edxsom",
			},
			expectedError: edxpackage.ErrIncorrectInputParam,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedError

			_, correct := edx.UseCase.PostEnrollment(testCase.message)
			assert.Equal(t, expect, correct)
		})
	}
}
