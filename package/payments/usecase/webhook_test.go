package usecase

import (
	"errors"
	"testing"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments/yookassa"
	"github.com/spf13/viper"
)

type fakeGateway struct {
	payments.Gateway
	orders   map[string]*models.OrderCore
	attempts []*models.PaymentAttemptCore
}

func (g *fakeGateway) GetOrderByNumber(n string) (*models.OrderCore, error) {
	for _, o := range g.orders {
		if o.OrderNumber == n {
			cp := *o
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (g *fakeGateway) GetOrderByPaymentID(id string) (*models.OrderCore, error) {
	for _, o := range g.orders {
		if o.YookassaPaymentID == id {
			cp := *o
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (g *fakeGateway) WithOrderLock(id string, fn func(*models.OrderCore) error) error {
	cp := *g.orders[id]
	if err := fn(&cp); err != nil {
		return err
	}
	g.orders[id] = &cp
	return nil
}

func (g *fakeGateway) GetProductByID(string) (*models.ProductCore, error) {
	return &models.ProductCore{ID: "prod-1", SeatLimit: 1, DurationDays: 30}, nil
}

func (g *fakeGateway) CreatePaymentAttempt(a *models.PaymentAttemptCore) (*models.PaymentAttemptCore, error) {
	g.attempts = append(g.attempts, a)
	return a, nil
}

type fakeLicensing struct {
	licensing.UseCase
	issued int
}

func (l *fakeLicensing) IssueLicense(models.IssueLicenseInput) (*models.LicenseCore, error) {
	l.issued++
	return &models.LicenseCore{ID: "lic-1"}, nil
}

type fakeYooKassa struct {
	yookassaAPI
	configured bool
	info       *yookassa.PaymentInfo
	err        error
	calls      int
}

func (f *fakeYooKassa) IsConfigured() bool { return f.configured }

func (f *fakeYooKassa) GetPayment(string) (*yookassa.PaymentInfo, error) {
	f.calls++
	return f.info, f.err
}

type webhookFixture struct {
	uc  *PaymentsUseCaseImpl
	gw  *fakeGateway
	lic *fakeLicensing
	yk  *fakeYooKassa
}

func newWebhookFixture(t *testing.T, apiStatus string) *webhookFixture {
	t.Helper()
	viper.Set("payments.webhookSkipIpCheck", true)
	t.Cleanup(func() { viper.Set("payments.webhookSkipIpCheck", false) })
	gw := &fakeGateway{orders: map[string]*models.OrderCore{
		"o1": {ID: "o1", OrderNumber: "ROBBO-1", ProductID: "prod-1", Status: models.OrderStatusPending, YookassaPaymentID: "pay-1"},
	}}
	yk := &fakeYooKassa{configured: true, info: &yookassa.PaymentInfo{
		ID: "pay-1", Status: apiStatus, Metadata: map[string]string{"order_number": "ROBBO-1"},
	}}
	lic := &fakeLicensing{}
	return &webhookFixture{uc: &PaymentsUseCaseImpl{gateway: gw, licensing: lic, yk: yk}, gw: gw, lic: lic, yk: yk}
}

func notification(event, paymentID, orderNumber string) []byte {
	return []byte(`{"event":"` + event + `","object":{"id":"` + paymentID + `","status":"succeeded","metadata":{"order_number":"` + orderNumber + `"}}}`)
}

func (f *webhookFixture) status() string { return f.gw.orders["o1"].Status }

func TestWebhookSucceededVerifiedIssuesLicenseOnce(t *testing.T) {
	f := newWebhookFixture(t, "succeeded")
	body := notification("payment.succeeded", "pay-1", "ROBBO-1")
	for i := 0; i < 2; i++ { // replayed notification must stay idempotent
		if err := f.uc.HandleWebhook(body, "185.71.76.1"); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if f.lic.issued != 1 || f.status() != models.OrderStatusPaid {
		t.Fatalf("issued=%d status=%s, want 1 license and paid", f.lic.issued, f.status())
	}
}

func TestWebhookForgedSucceededWhilePaymentPending(t *testing.T) {
	f := newWebhookFixture(t, "pending")
	err := f.uc.HandleWebhook(notification("payment.succeeded", "pay-1", "ROBBO-1"), "185.71.76.1")
	if !errors.Is(err, payments.ErrPaymentVerificationUnavailable) {
		t.Fatalf("err=%v want ErrPaymentVerificationUnavailable", err)
	}
	if f.lic.issued != 0 || f.status() != models.OrderStatusPending {
		t.Fatalf("issued=%d status=%s, want no license and pending", f.lic.issued, f.status())
	}
}

func TestWebhookPaymentIDMismatchRejectedWithoutAPICall(t *testing.T) {
	f := newWebhookFixture(t, "succeeded")
	err := f.uc.HandleWebhook(notification("payment.succeeded", "pay-other", "ROBBO-1"), "185.71.76.1")
	if !errors.Is(err, payments.ErrBadRequest) {
		t.Fatalf("err=%v want ErrBadRequest", err)
	}
	if f.lic.issued != 0 || f.yk.calls != 0 {
		t.Fatalf("issued=%d apiCalls=%d, want none", f.lic.issued, f.yk.calls)
	}
}

func TestWebhookSucceededButAPICanceled(t *testing.T) {
	f := newWebhookFixture(t, "canceled")
	err := f.uc.HandleWebhook(notification("payment.succeeded", "pay-1", "ROBBO-1"), "185.71.76.1")
	if !errors.Is(err, payments.ErrBadRequest) || f.lic.issued != 0 {
		t.Fatalf("err=%v issued=%d, want ErrBadRequest and no license", err, f.lic.issued)
	}
}

func TestWebhookForgedCancelKeepsOrderPending(t *testing.T) {
	f := newWebhookFixture(t, "succeeded")
	err := f.uc.HandleWebhook(notification("payment.canceled", "pay-1", "ROBBO-1"), "185.71.76.1")
	if !errors.Is(err, payments.ErrBadRequest) || f.status() != models.OrderStatusPending {
		t.Fatalf("err=%v status=%s, want ErrBadRequest and pending", err, f.status())
	}
}

func TestWebhookCancelVerified(t *testing.T) {
	f := newWebhookFixture(t, "canceled")
	if err := f.uc.HandleWebhook(notification("payment.canceled", "pay-1", "ROBBO-1"), "185.71.76.1"); err != nil {
		t.Fatal(err)
	}
	if f.status() != models.OrderStatusCanceled {
		t.Fatalf("status=%s want canceled", f.status())
	}
}

func TestWebhookWithoutYooKassaConfigDoesNotFulfil(t *testing.T) {
	f := newWebhookFixture(t, "succeeded")
	f.yk.configured = false
	err := f.uc.HandleWebhook(notification("payment.succeeded", "pay-1", "ROBBO-1"), "185.71.76.1")
	if !errors.Is(err, payments.ErrPaymentNotConfigured) || f.lic.issued != 0 {
		t.Fatalf("err=%v issued=%d, want ErrPaymentNotConfigured and no license", err, f.lic.issued)
	}
}

func TestWebhookUntrustedSourceRejected(t *testing.T) {
	f := newWebhookFixture(t, "succeeded")
	viper.Set("payments.webhookSkipIpCheck", false)
	err := f.uc.HandleWebhook(notification("payment.succeeded", "pay-1", "ROBBO-1"), "203.0.113.7")
	if !errors.Is(err, payments.ErrUntrustedWebhookSource) || f.lic.issued != 0 {
		t.Fatalf("err=%v issued=%d, want ErrUntrustedWebhookSource", err, f.lic.issued)
	}
}
