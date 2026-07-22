package payments

import "errors"

var (
	ErrProductNotFound          = errors.New("PRODUCT_NOT_FOUND")
	ErrOrderNotFound            = errors.New("ORDER_NOT_FOUND")
	ErrPaymentNotConfigured     = errors.New("PAYMENT_NOT_CONFIGURED")
	ErrUntrustedWebhookSource   = errors.New("UNTRUSTED_WEBHOOK_SOURCE")
	ErrFulfillmentFailed        = errors.New("FULFILLMENT_FAILED")
	ErrBadRequest               = errors.New("bad_request")
	ErrForbidden                = errors.New("FORBIDDEN")
	ErrYookassaPaymentFailed    = errors.New("YOOKASSA_PAYMENT_FAILED")
)
