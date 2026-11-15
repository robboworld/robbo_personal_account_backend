package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/auth"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
)

type kidTestAuth struct{ auth.Delegate }

func (kidTestAuth) UserIdentity(*gin.Context) (string, models.Role, error) {
	return "5", models.Student, nil
}

type kidTestPayments struct {
	payments.Delegate
	called string
}

func (p *kidTestPayments) Checkout(string, string, string) (string, *models.OrderCore, error) {
	p.called = "Checkout"
	return "https://pay.example/confirm", &models.OrderCore{}, nil
}

func (p *kidTestPayments) SyncOrder(string, string, string) (*models.OrderCore, error) {
	p.called = "SyncOrder"
	return &models.OrderCore{}, nil
}

var kidLockedPaymentRoutes = []struct{ method, path, body string }{
	{http.MethodPost, "/payments/checkout", `{"productId":"pro-month"}`},
	{http.MethodGet, "/payments/orders/ORD-1", ""},
}

func callPaymentRoute(t *testing.T, email, method, path, body string) (*httptest.ResponseRecorder, *kidTestPayments) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.MarkKidAccountSession(c, email)
		c.Next()
	})
	p := &kidTestPayments{}
	h := &Handler{authDelegate: kidTestAuth{}, paymentsDelegate: p}
	h.InitPaymentsRoutes(router)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, p
}

// A kid account buys nothing (architecture/ADR_CLASS_KID_LOGIN.md): 403 kid_account, and
// no order is created or synced.
func TestPaymentsRefuseKidAccount(t *testing.T) {
	for _, r := range kidLockedPaymentRoutes {
		rec, p := callPaymentRoute(t, "k-abc@kids.robbo.invalid", r.method, r.path, r.body)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "kid_account") {
			t.Errorf("%s %s: want 403 kid_account, got %d %s", r.method, r.path, rec.Code, rec.Body.String())
		}
		if p.called != "" {
			t.Errorf("%s %s: kid account reached %s", r.method, r.path, p.called)
		}
	}
}

func TestPaymentsKeepFullAccount(t *testing.T) {
	for _, r := range kidLockedPaymentRoutes {
		rec, p := callPaymentRoute(t, "parent@example.com", r.method, r.path, r.body)
		if rec.Code != http.StatusOK || p.called == "" {
			t.Errorf("%s %s: full account: want 200 via the use case, got %d %s", r.method, r.path, rec.Code, rec.Body.String())
		}
	}
}
