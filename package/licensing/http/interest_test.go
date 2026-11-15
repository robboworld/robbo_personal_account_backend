package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
)

type fakeAuth struct {
	auth.Delegate
	id   string
	role models.Role
	err  error
}

func (a fakeAuth) UserIdentity(*gin.Context) (string, models.Role, error) {
	return a.id, a.role, a.err
}

type fakeLicensing struct {
	licensing.Delegate
	rows   []models.FeatureInterestCore
	total  int64
	err    error
	limit  int
	offset int
	called bool
}

func (l *fakeLicensing) ListFeatureInterest(limit, offset int) ([]models.FeatureInterestCore, int64, error) {
	l.called = true
	l.limit, l.offset = limit, offset
	return l.rows, l.total, l.err
}

// serve routes GET /licensing/interest/answers through the real route table.
func serve(t *testing.T, a fakeAuth, l *fakeLicensing, query string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &Handler{authDelegate: a, licensingDelegate: l}
	h.InitLicensingRoutes(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/licensing/interest/answers"+query, nil))
	return rec
}

func TestListInterestAnswersOnlyForSuperAdmin(t *testing.T) {
	for _, role := range []models.Role{models.Student, models.Teacher, models.Parent, models.FreeListener, models.UnitAdmin} {
		l := &fakeLicensing{}
		rec := serve(t, fakeAuth{id: "7", role: role}, l, "")
		if rec.Code != http.StatusForbidden {
			t.Errorf("role %d: want 403, got %d", role, rec.Code)
		}
		if l.called {
			t.Errorf("role %d: the answers were read before the role check", role)
		}
	}
}

func TestListInterestAnswersUnauthorized(t *testing.T) {
	l := &fakeLicensing{}
	rec := serve(t, fakeAuth{err: errors.New("no session")}, l, "")
	if rec.Code != http.StatusUnauthorized || l.called {
		t.Fatalf("want 401 without reading answers, got %d called=%v", rec.Code, l.called)
	}
}

func TestListInterestAnswersReturnsAnswersWithoutAccountID(t *testing.T) {
	when := time.Date(2026, 11, 1, 10, 30, 0, 0, time.UTC)
	l := &fakeLicensing{total: 3, rows: []models.FeatureInterestCore{{
		LmsUserID: "secret-account-42", Role: "club_owner", Feature: "other", Other: "reports",
		Pain: "too much paperwork", Org: "ROBBO, Kazan", Notify: true, Source: "landing", UpdatedAt: when,
	}}}
	rec := serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, l, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body)
	}
	if got := rec.Body.String(); strings.Contains(got, "secret-account-42") || strings.Contains(got, "lms_user_id") || strings.Contains(got, "lmsUserId") {
		t.Fatalf("account id leaked into the response: %s", got)
	}
	var body struct {
		Total   int64                    `json:"total"`
		Answers []map[string]interface{} `json:"answers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 3 || len(body.Answers) != 1 {
		t.Fatalf("unexpected body: %s", rec.Body)
	}
	a := body.Answers[0]
	want := map[string]interface{}{
		"role": "club_owner", "feature": "other", "other": "reports", "pain": "too much paperwork",
		"org": "ROBBO, Kazan", "notify": true, "source": "landing", "updatedAt": "2026-11-01T10:30:00Z",
	}
	for k, v := range want {
		if a[k] != v {
			t.Errorf("%s: want %v, got %v", k, v, a[k])
		}
	}
}

func TestListInterestAnswersEmptyIsArrayNotNull(t *testing.T) {
	rec := serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, &fakeLicensing{}, "")
	if !strings.Contains(rec.Body.String(), `"answers":[]`) {
		t.Fatalf("empty list must be [], got %s", rec.Body)
	}
}

func TestListInterestAnswersPagingParams(t *testing.T) {
	l := &fakeLicensing{}
	serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, l, "?limit=20&offset=40")
	if l.limit != 20 || l.offset != 40 {
		t.Fatalf("got limit=%d offset=%d", l.limit, l.offset)
	}
	l = &fakeLicensing{}
	serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, l, "")
	if l.limit != 50 || l.offset != 0 {
		t.Fatalf("defaults: got limit=%d offset=%d", l.limit, l.offset)
	}
	l = &fakeLicensing{}
	serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, l, "?limit=abc&offset=xyz")
	// Unparsable numbers reach the usecase as 0; it clamps them to the default page (usecase test).
	if l.limit != 0 || l.offset != 0 {
		t.Fatalf("garbage params: got limit=%d offset=%d", l.limit, l.offset)
	}
}

func TestListInterestAnswersStorageErrorIs500(t *testing.T) {
	rec := serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, &fakeLicensing{err: errors.New("db down")}, "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
}

// The static /interest/answers route must not be swallowed by /:licenseId.
func TestInterestAnswersRouteIsNotALicenseID(t *testing.T) {
	l := &fakeLicensing{}
	serve(t, fakeAuth{id: "1", role: models.SuperAdmin}, l, "")
	if !l.called {
		t.Fatal("request did not reach ListFeatureInterestAnswers")
	}
}
