//go:build external

// Calls the live edx test API; run with: go test -tags external ./package/edx/...
package courses

import (
	"errors"
	"github.com/go-playground/assert/v2"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/config"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx/usecase"
	"log"
	"testing"
)

func TestEdxApiUseCaseImpl_GetCourseContent(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edxUseCase := usecase.SetupEdxApiUseCase()
	testTable := []struct {
		name          string
		courseId      string
		expectedError error
	}{
		{
			name:          "Ok",
			courseId:      "course-v1:Test_org+01+2022",
			expectedError: nil,
		},

		{
			name:          "Bad courseId",
			courseId:      "Ddssadad",
			expectedError: edx.ErrIncorrectInputParam,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			expect := testCase.expectedError
			_, correct := edxUseCase.ApiCourse.GetCourseContent(testCase.courseId)
			assert.Equal(t, expect, correct)
		})
	}
}

func TestEdxApiUseCaseImpl_GetEnrollments(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edx := usecase.SetupEdxApiUseCase()
	testTable := []struct {
		name         string
		username     string
		expectedBody string
	}{
		{
			name:         "Ok",
			username:     "edxsom",
			expectedBody: "{\"next\":null,\"previous\":null,\"results\":[{\"created\":\"2022-06-13T03:00:12.571664Z\",\"mode\":\"honor\",\"is_active\":true,\"user\":\"edxsom\",\"course_id\":\"course-v1:TestOrg+02+2022\"},{\"created\":\"2022-06-13T01:16:45.374794Z\",\"mode\":\"honor\",\"is_active\":true,\"user\":\"edxsom\",\"course_id\":\"course-v1:Test_org+01+2022\"}]}",
		},

		{
			name:         "Bad username",
			username:     "dsad",
			expectedBody: "{\"next\":null,\"previous\":null,\"results\":[]}",
		},
		{
			name:         "Empty username",
			username:     "",
			expectedBody: "{\"next\":null,\"previous\":null,\"results\":[{\"created\":\"2022-07-12T14:21:28.212240Z\",\"mode\":\"audit\",\"is_active\":true,\"user\":\"vovantyarus\",\"course_id\":\"course-v1:edX+DemoX+Demo_Course\"},{\"created\":\"2022-06-18T21:59:34.558581Z\",\"mode\":\"audit\",\"is_active\":true,\"user\":\"tesr_user\",\"course_id\":\"course-v1:Test_org+01+2022\"},{\"created\":\"2022-06-13T03:00:12.571664Z\",\"mode\":\"honor\",\"is_active\":true,\"user\":\"edxsom\",\"course_id\":\"course-v1:TestOrg+02+2022\"},{\"created\":\"2022-06-13T01:16:45.374794Z\",\"mode\":\"honor\",\"is_active\":true,\"user\":\"edxsom\",\"course_id\":\"course-v1:Test_org+01+2022\"}]}",
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedBody

			correct, _ := edx.ApiCourse.GetEnrollments(testCase.username)
			assert.Equal(t, expect, string(correct))
		})
	}
}

func TestEdxApiUseCaseImpl_GetAllPublicCourses(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edxUseCase := usecase.SetupEdxApiUseCase()
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
			expectedError: errors.New("user not found"),
		},
		{
			name:          "Page number more then page count",
			pageNumber:    423423423,
			expectedError: errors.New("user not found"),
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedError

			_, correct := edxUseCase.ApiCourse.GetAllPublicCourses(testCase.pageNumber)
			assert.Equal(t, expect, correct)
		})
	}
}

func TestEdxApiUseCaseImpl_PostEnrollment(t *testing.T) {
	if err := config.InitForTests(); err != nil {
		log.Fatalf("%s", err.Error())
	}
	edxUseCase := usecase.SetupEdxApiUseCase()
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
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name: "Username incorrect",
			message: map[string]interface{}{
				"course_details": map[string]string{
					"course_id": "course-v1:Test_org+01+2022",
				},
				"user": "edm",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
		{
			name: "Empty field courseId",
			message: map[string]interface{}{
				"course_details": map[string]string{
					"course_id": "",
				},
				"user": "edxsom",
			},
			expectedError: edx.ErrIncorrectInputParam,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {

			expect := testCase.expectedError

			_, correct := edxUseCase.ApiCourse.PostEnrollment(testCase.message)
			assert.Equal(t, expect, correct)
		})
	}
}
