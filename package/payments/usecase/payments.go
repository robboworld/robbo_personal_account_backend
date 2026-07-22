package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/licensing"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments/yookassa"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type PaymentsUseCaseImpl struct {
	gateway   payments.Gateway
	licensing licensing.UseCase
	yk        *yookassa.Client
}

type PaymentsUseCaseModule struct {
	fx.Out
	payments.UseCase
}

func SetupPaymentsUseCase(gateway payments.Gateway, licensingUC licensing.UseCase) PaymentsUseCaseModule {
	return PaymentsUseCaseModule{
		UseCase: &PaymentsUseCaseImpl{
			gateway:   gateway,
			licensing: licensingUC,
			yk:        yookassa.NewClientFromViper(),
		},
	}
}

func (u *PaymentsUseCaseImpl) ListActiveProducts() ([]*models.ProductCore, error) {
	return u.gateway.ListActiveProducts()
}

func (u *PaymentsUseCaseImpl) CreateProduct(input models.CreateProductInput) (*models.ProductCore, error) {
	if strings.TrimSpace(input.SKU) == "" || strings.TrimSpace(input.Title) == "" || input.Amount <= 0 {
		return nil, payments.ErrBadRequest
	}
	currency := input.Currency
	if currency == "" {
		currency = "RUB"
	}
	seatLimit := input.SeatLimit
	if seatLimit <= 0 {
		seatLimit = 1
	}
	duration := input.DurationDays
	if duration <= 0 {
		duration = 365
	}
	caps := input.Capabilities
	if len(caps) == 0 {
		caps = []string{models.CapabilityPremiumAuto}
	}
	return u.gateway.CreateProduct(&models.ProductCore{
		SKU:          strings.TrimSpace(input.SKU),
		Title:        strings.TrimSpace(input.Title),
		Description:  input.Description,
		Amount:       input.Amount,
		Currency:     currency,
		SeatLimit:    seatLimit,
		Capabilities: caps,
		DurationDays: duration,
		IsActive:     input.IsActive,
	})
}

func (u *PaymentsUseCaseImpl) Checkout(lmsUserID, productID, customerEmail string) (string, *models.OrderCore, error) {
	if lmsUserID == "" || productID == "" {
		return "", nil, payments.ErrBadRequest
	}
	if !u.yk.IsConfigured() {
		return "", nil, payments.ErrPaymentNotConfigured
	}
	product, err := u.gateway.GetProductByID(productID)
	if err != nil {
		return "", nil, err
	}
	if !product.IsActive {
		return "", nil, payments.ErrProductNotFound
	}

	orderNumber, err := newOrderNumber()
	if err != nil {
		return "", nil, err
	}
	idempotencyKey, err := newIdempotencyKey()
	if err != nil {
		return "", nil, err
	}
	order, err := u.gateway.CreateOrder(&models.OrderCore{
		OrderNumber:    orderNumber,
		LmsUserID:      lmsUserID,
		ProductID:      product.ID,
		Amount:         product.Amount,
		Currency:       product.Currency,
		Status:         models.OrderStatusPending,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return "", nil, err
	}

	returnURL := strings.TrimRight(viper.GetString("payments.returnUrlBase"), "/")
	if returnURL == "" {
		returnURL = "http://localhost:3030/licenses/receipt"
	}
	returnURL = fmt.Sprintf("%s?order_number=%s", returnURL, order.OrderNumber)

	receipt := yookassa.BuildReceipt(product.Amount, product.Currency, product.Title, customerEmail)
	result, err := u.yk.CreatePayment(yookassa.CreatePaymentRequest{
		IdempotencyKey: order.IdempotencyKey,
		Amount:         product.Amount,
		Currency:       product.Currency,
		Description:    product.Title,
		ReturnURL:      returnURL,
		Metadata: map[string]string{
			"order_number": order.OrderNumber,
			"product_id":   product.ID,
			"lms_user_id":  lmsUserID,
		},
		Receipt: receipt,
	})
	if err != nil {
		_ = u.gateway.UpdateOrder(&models.OrderCore{
			ID:     order.ID,
			Status: models.OrderStatusFailed,
		})
		return "", nil, fmt.Errorf("%w: %v", payments.ErrYookassaPaymentFailed, err)
	}

	order.YookassaPaymentID = result.PaymentID
	if err := u.gateway.UpdateOrder(order); err != nil {
		return "", nil, err
	}
	return result.ConfirmationURL, order, nil
}

func (u *PaymentsUseCaseImpl) GetOrder(lmsUserID, orderNumber string) (*models.OrderCore, error) {
	order, err := u.gateway.GetOrderByNumber(orderNumber)
	if err != nil {
		return nil, err
	}
	if order.LmsUserID != lmsUserID {
		return nil, payments.ErrForbidden
	}
	return order, nil
}

func (u *PaymentsUseCaseImpl) SyncOrder(lmsUserID, orderNumber, sourceIP string) (*models.OrderCore, error) {
	order, err := u.GetOrder(lmsUserID, orderNumber)
	if err != nil {
		return nil, err
	}
	if order.Status != models.OrderStatusPending || order.YookassaPaymentID == "" {
		return order, nil
	}
	if !u.yk.IsConfigured() {
		return order, nil
	}
	info, err := u.yk.GetPayment(order.YookassaPaymentID)
	if err != nil {
		return order, nil
	}
	switch info.Status {
	case "succeeded":
		_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
			OrderID:   order.ID,
			EventType: "payment.succeeded",
			Status:    models.PaymentAttemptReceived,
			SourceIP:  sourceIP,
			Payload: map[string]interface{}{
				"payment_id": order.YookassaPaymentID,
				"source":     "receipt_sync",
			},
		})
		if err := u.fulfillOrder(order.ID); err != nil {
			_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
				OrderID:   order.ID,
				EventType: "payment.succeeded",
				Status:    models.PaymentAttemptError,
				SourceIP:  sourceIP,
				Payload: map[string]interface{}{
					"payment_id": order.YookassaPaymentID,
					"source":     "receipt_sync",
					"error":      err.Error(),
				},
			})
			return u.gateway.GetOrderByID(order.ID)
		}
		_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
			OrderID:   order.ID,
			EventType: "payment.succeeded",
			Status:    models.PaymentAttemptSucceeded,
			SourceIP:  sourceIP,
			Payload: map[string]interface{}{
				"payment_id": order.YookassaPaymentID,
				"source":     "receipt_sync",
			},
		})
	case "canceled":
		order.Status = models.OrderStatusCanceled
		_ = u.gateway.UpdateOrder(order)
		_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
			OrderID:   order.ID,
			EventType: "payment.canceled",
			Status:    models.PaymentAttemptCanceled,
			SourceIP:  sourceIP,
			Payload: map[string]interface{}{
				"payment_id": order.YookassaPaymentID,
				"source":     "receipt_sync",
			},
		})
	}
	return u.gateway.GetOrderByID(order.ID)
}

func (u *PaymentsUseCaseImpl) HandleWebhook(rawBody []byte, sourceIP string) error {
	skipIP := viper.GetBool("payments.webhookSkipIpCheck")
	if !skipIP && !yookassa.IsTrustedIP(sourceIP) {
		return payments.ErrUntrustedWebhookSource
	}
	event, paymentID, metadata, err := yookassa.ParseWebhookNotification(rawBody)
	if err != nil {
		return payments.ErrBadRequest
	}

	var order *models.OrderCore
	if orderNumber := metadata["order_number"]; orderNumber != "" {
		order, _ = u.gateway.GetOrderByNumber(orderNumber)
	}
	if order == nil && paymentID != "" {
		order, _ = u.gateway.GetOrderByPaymentID(paymentID)
	}
	if order == nil {
		// Unknown order: acknowledge silently (same as LMS reference).
		return nil
	}

	_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
		OrderID:   order.ID,
		EventType: event,
		Status:    models.PaymentAttemptReceived,
		SourceIP:  sourceIP,
		Payload: map[string]interface{}{
			"event":      event,
			"payment_id": paymentID,
		},
	})

	switch event {
	case "payment.succeeded":
		if err := u.fulfillOrder(order.ID); err != nil {
			_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
				OrderID:   order.ID,
				EventType: event,
				Status:    models.PaymentAttemptError,
				SourceIP:  sourceIP,
				Payload: map[string]interface{}{
					"payment_id": paymentID,
					"error":      err.Error(),
				},
			})
			return payments.ErrFulfillmentFailed
		}
		_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
			OrderID:   order.ID,
			EventType: event,
			Status:    models.PaymentAttemptSucceeded,
			SourceIP:  sourceIP,
			Payload:   map[string]interface{}{"payment_id": paymentID},
		})
	case "payment.canceled":
		_ = u.gateway.WithOrderLock(order.ID, func(locked *models.OrderCore) error {
			if locked.Status == models.OrderStatusPending {
				locked.Status = models.OrderStatusCanceled
			}
			return nil
		})
		_, _ = u.gateway.CreatePaymentAttempt(&models.PaymentAttemptCore{
			OrderID:   order.ID,
			EventType: event,
			Status:    models.PaymentAttemptCanceled,
			SourceIP:  sourceIP,
			Payload:   map[string]interface{}{"payment_id": paymentID},
		})
	}
	return nil
}

func (u *PaymentsUseCaseImpl) fulfillOrder(orderID string) error {
	return u.gateway.WithOrderLock(orderID, func(order *models.OrderCore) error {
		if order.Status == models.OrderStatusPaid {
			return nil
		}
		if order.Status != models.OrderStatusPending {
			return fmt.Errorf("%w: order status %s", payments.ErrFulfillmentFailed, order.Status)
		}
		product, err := u.gateway.GetProductByID(order.ProductID)
		if err != nil {
			return err
		}
		expiresAt := time.Now().UTC().AddDate(0, 0, product.DurationDays)
		lic, err := u.licensing.IssueLicense(models.IssueLicenseInput{
			LmsUserID:    order.LmsUserID,
			SeatLimit:    product.SeatLimit,
			Capabilities: product.Capabilities,
			ExpiresAt:    expiresAt,
			Note:         fmt.Sprintf("order %s", order.OrderNumber),
			ProductID:    product.ID,
			Source:       models.LicenseSourceOrder,
		})
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		order.Status = models.OrderStatusPaid
		order.LicenseID = lic.ID
		order.PaidAt = &now
		return nil
	})
}

func newOrderNumber() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ROBBO-" + strings.ToUpper(hex.EncodeToString(b)), nil
}

func newIdempotencyKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
