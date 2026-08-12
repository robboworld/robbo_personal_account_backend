package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/edx"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (p *EdxApiUseCaseImpl) GetWithAuth(url string) (respBody []byte, err error) {
	err = p.RefreshToken()

	if err != nil {
		log.Println("Token not refresh.\n[ERROR] -", err)
		return nil, edx.ErrTknNotRefresh
	}
	var bearer = "Bearer " + viper.GetString("api.token")

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Error on request.\n[ERROR] -", err)
		return nil, edx.ErrOnReq
	}

	request.Header.Add("Authorization", bearer)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
		return nil, edx.ErrOnResp
	}
	if response.StatusCode != http.StatusOK {
		return nil, edx.ErrIncorrectInputParam
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
		return nil, edx.ErrReadRespBody
	}
	return body, nil
}

func (p *EdxApiUseCaseImpl) PostWithAuth(url string, params map[string]interface{}) (respBody []byte, err error) {
	err = p.RefreshToken()
	if err != nil {
		log.Println("token not refresh")
		return nil, edx.ErrTknNotRefresh

	}

	data, err := json.Marshal(params)

	if err != nil {
		log.Println(err)
		return nil, edx.ErrJsonMarshal
	}

	var bearer = "Bearer " + viper.GetString("api.token")

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
		return nil, edx.ErrOnReq
	}

	request.Header.Add("Authorization", bearer)
	request.Header.Add("Content-Type", "application/json;charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrOnResp
	}
	if response.StatusCode != http.StatusOK {
		return nil, edx.ErrIncorrectInputParam
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrReadRespBody
	}
	return body, nil
}

func (p *EdxApiUseCaseImpl) RefreshToken() (err error) {
	if viper.GetInt64("api.token_expiration_time") < time.Now().Unix() {
		urlAddr := viper.GetString("api_urls.refreshToken")
		response, err := http.PostForm(urlAddr, url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {viper.GetString("api.client_id")},
			"client_secret": {viper.GetString("api.client_secret")},
		})
		if err != nil {
			log.Println(err)
			return edx.ErrOnReq
		}
		if response.StatusCode != http.StatusOK {
			return edx.ErrIncorrectInputParam
		}

		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println(err)
			return edx.ErrIncorrectInputParam
		}

		newtkn := &edx.NewToken{}
		err = json.Unmarshal(body, newtkn)
		if err != nil {
			log.Println(err)
			return errors.New("Error on json unmarshal")
		}

		expirationTime := time.Now().Unix() + int64(newtkn.ExpiresIn)
		viper.Set("api.token_expiration_time", expirationTime)
		viper.Set("api.token", newtkn.AccessToken)
		return nil
	} else {
		return nil
	}
}

func (p *EdxApiUseCaseImpl) CreateCohort(courseId string, cohortParams map[string]interface{}) (respBody []byte, err error) {
	urlAddr := viper.GetString("api_urls.postCohort") + courseId + "/cohorts/"
	return p.PostWithAuth(urlAddr, cohortParams)
}

func (p *EdxApiUseCaseImpl) AddStudent(username, courseId string, cohortId int) (respBody []byte, err error) {
	err = p.RefreshToken()
	if err != nil {
		log.Println("token not refresh")
		return nil, edx.ErrTknNotRefresh

	}

	var bearer = "Bearer " + viper.GetString("api.token")
	urlAddr := viper.GetString("api_urls.postCohort") + courseId + "/cohorts/" + strconv.Itoa(cohortId) + "/users/" + username
	request, err := http.NewRequest("POST", urlAddr, nil)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrOnReq
	}

	request.Header.Add("Authorization", bearer)
	request.Header.Add("Content-Type", "application/json;charset=utf-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrOnResp
	}
	if response.StatusCode != http.StatusOK {
		return nil, edx.ErrIncorrectInputParam
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrReadRespBody
	}
	return body, nil
}

func cohortAPIRoot() string {
	base := viper.GetString("api_urls.postCohort")
	const suffix = "courses/"
	if len(base) >= len(suffix) && base[len(base)-len(suffix):] == suffix {
		return base[:len(base)-len(suffix)]
	}
	return base
}

func encodeCourseKey(courseId string) string {
	return url.PathEscape(courseId)
}

func (p *EdxApiUseCaseImpl) requestWithAuth(method, urlAddr string, params map[string]interface{}, okStatuses ...int) (respBody []byte, err error) {
	err = p.RefreshToken()
	if err != nil {
		return nil, edx.ErrTknNotRefresh
	}
	var bodyReader *bytes.Buffer
	if params != nil {
		data, mErr := json.Marshal(params)
		if mErr != nil {
			return nil, edx.ErrJsonMarshal
		}
		bodyReader = bytes.NewBuffer(data)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}
	request, err := http.NewRequest(method, urlAddr, bodyReader)
	if err != nil {
		return nil, edx.ErrOnReq
	}
	request.Header.Add("Authorization", "Bearer "+viper.GetString("api.token"))
	if params != nil {
		request.Header.Add("Content-Type", "application/json;charset=utf-8")
	}
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, edx.ErrOnResp
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, edx.ErrReadRespBody
	}
	if len(okStatuses) == 0 {
		okStatuses = []int{http.StatusOK}
	}
	for _, code := range okStatuses {
		if response.StatusCode == code {
			return body, nil
		}
	}
	log.Printf("edx %s %s -> %d: %s", method, urlAddr, response.StatusCode, string(body))
	return nil, edx.ErrIncorrectInputParam
}

func (p *EdxApiUseCaseImpl) GetCohortSettings(courseId string) (respBody []byte, err error) {
	urlAddr := cohortAPIRoot() + "settings/" + encodeCourseKey(courseId)
	return p.GetWithAuth(urlAddr)
}

func (p *EdxApiUseCaseImpl) SetCohortSettings(courseId string, params map[string]interface{}) (respBody []byte, err error) {
	urlAddr := cohortAPIRoot() + "settings/" + encodeCourseKey(courseId)
	return p.requestWithAuth(http.MethodPut, urlAddr, params, http.StatusOK)
}

func (p *EdxApiUseCaseImpl) ListCohorts(courseId string) (respBody []byte, err error) {
	urlAddr := viper.GetString("api_urls.postCohort") + encodeCourseKey(courseId) + "/cohorts/"
	return p.GetWithAuth(urlAddr)
}

func (p *EdxApiUseCaseImpl) GetCohort(courseId string, cohortId int) (respBody []byte, err error) {
	urlAddr := viper.GetString("api_urls.postCohort") + encodeCourseKey(courseId) + "/cohorts/" + strconv.Itoa(cohortId)
	return p.GetWithAuth(urlAddr)
}

func (p *EdxApiUseCaseImpl) PatchCohort(courseId string, cohortId int, params map[string]interface{}) (respBody []byte, err error) {
	urlAddr := viper.GetString("api_urls.postCohort") + encodeCourseKey(courseId) + "/cohorts/" + strconv.Itoa(cohortId)
	return p.requestWithAuth(http.MethodPatch, urlAddr, params, http.StatusOK)
}

func (p *EdxApiUseCaseImpl) ListCohortUsers(courseId string, cohortId int) (respBody []byte, err error) {
	urlAddr := viper.GetString("api_urls.postCohort") + encodeCourseKey(courseId) + "/cohorts/" + strconv.Itoa(cohortId) + "/users"
	return p.GetWithAuth(urlAddr)
}

func (p *EdxApiUseCaseImpl) RemoveCohortUser(username, courseId string, cohortId int) (err error) {
	urlAddr := viper.GetString("api_urls.postCohort") + encodeCourseKey(courseId) + "/cohorts/" + strconv.Itoa(cohortId) + "/users/" + url.PathEscape(username)
	_, err = p.requestWithAuth(http.MethodDelete, urlAddr, nil, http.StatusOK, http.StatusNoContent)
	return err
}

func (p *EdxApiUseCaseImpl) GetAllPublicCourses(pageNumber int) (respBody []byte, err error) {
	if pageNumber <= 0 || pageNumber >= 5000 {
		return nil, edx.ErrOnReq
	}
	resp, err := http.Get(viper.GetString("api_urls.getAllPublicCourses") + strconv.Itoa(pageNumber) + "&page_size=5")
	if err != nil {
		log.Println(err)
		return nil, edx.ErrOnReq
	}
	if resp.StatusCode != http.StatusOK {
		return nil, edx.ErrIncorrectInputParam
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrReadRespBody
	}
	return body, nil
}

func (p *EdxApiUseCaseImpl) GetCoursesByUser() (respBody []byte, err error) {
	response, err := http.Get(viper.GetString("api_urls.getCourses"))
	if err != nil {
		log.Println(err)
		return nil, edx.ErrOnReq
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return nil, edx.ErrReadRespBody
	}

	return body, nil
}

func (p *EdxApiUseCaseImpl) GetEnrollments(username string) (respBody []byte, err error) {
	return p.GetWithAuth(viper.GetString("api_urls.getEnrollment") + username)
}

func (p *EdxApiUseCaseImpl) GetUser() (respBody []byte, err error) {
	return p.GetWithAuth(viper.GetString("api_urls.getUser"))
}

func (p *EdxApiUseCaseImpl) GetCourseContent(courseId string) (respBody []byte, err error) {
	return p.GetWithAuth(viper.GetString("api_urls.getCourse") + courseId)
}

func (p *EdxApiUseCaseImpl) PostEnrollment(message map[string]interface{}) (respBody []byte, err error) {
	return p.PostWithAuth(viper.GetString("api_urls.postEnrollment"), message)
}

func (p *EdxApiUseCaseImpl) PostRegistration(registrationMessage edx.RegistrationForm) (respBody []byte, err error) {

	urlAddr := viper.GetString("api_urls.postRegistration")

	client := &http.Client{}
	jar := &myjar{}
	jar.jar = make(map[string][]*http.Cookie)
	client.Jar = jar

	resp, err := client.Get(viper.GetString("api_urls.getRegistration"))
	token, flag := handleCookies(resp.Cookies())
	if flag == false {
		return nil, errors.New("csrf token not found")
	}

	buffer := new(bytes.Buffer)
	params := url.Values{}
	params.Set("email", registrationMessage.Email)
	params.Set("password", registrationMessage.Password)
	params.Set("name", registrationMessage.Name)
	params.Set("username", registrationMessage.Username)
	params.Set("terms_of_service", registrationMessage.TermsOfService)
	buffer.WriteString(params.Encode())

	request, err := http.NewRequest("POST", urlAddr, buffer)
	request.Header.Add("x-csrftoken", token)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Add("Referer", "https://edx-test.ru/login?next=%2F'")
	resp, err = client.Do(request)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Error while reading the response bytes")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, edx.ErrIncorrectInputParam
	}
	return body, nil
}

func (p *EdxApiUseCaseImpl) Login(email, password string) (respBody []byte, err error) {

	urlAddr := viper.GetString("api_urls.login")
	client := &http.Client{}
	jar := &myjar{}
	jar.jar = make(map[string][]*http.Cookie)
	client.Jar = jar

	resp, err := client.Get(viper.GetString("api_urls.getLogin"))
	token, flag := handleCookies(resp.Cookies())
	if flag == false {
		return nil, errors.New("csrf token not found")
	}

	buffer := new(bytes.Buffer)
	params := url.Values{}
	params.Set("email", email)
	params.Set("password", password)
	buffer.WriteString(params.Encode())

	request, err := http.NewRequest("POST", urlAddr, buffer)
	request.Header.Add("x-csrftoken", token)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Add("Referer", "https://edx-test.ru/login?next=%2F'")
	resp, err = client.Do(request)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fmt.Println(string(body))
	if resp.StatusCode != http.StatusOK {
		return nil, edx.ErrIncorrectInputParam
	}
	return body, nil

}

type myjar struct {
	jar map[string][]*http.Cookie
}

func (p *myjar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	p.jar[u.Host] = cookies
}

func (p *myjar) Cookies(u *url.URL) []*http.Cookie {
	return p.jar[u.Host]
}

func handleCookies(n []*http.Cookie) (csrfToken string, found bool) {
	for _, cookie := range n {
		if cookie.Name == "csrftoken" {
			return cookie.Value, true
		}
	}
	return "", false
}

type EdxApiUseCaseImpl struct {
}

type EdxApiUseCaseModule struct {
	fx.Out
	edx.UseCase
}

func SetupEdxApiUseCase() EdxApiUseCaseModule {
	return EdxApiUseCaseModule{
		UseCase: &EdxApiUseCaseImpl{},
	}
}
