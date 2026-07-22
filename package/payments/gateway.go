package payments

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

// Gateway persists products, orders and payment attempts.
type Gateway interface {
	ListActiveProducts() ([]*models.ProductCore, error)
	GetProductByID(id string) (*models.ProductCore, error)
	CreateProduct(product *models.ProductCore) (*models.ProductCore, error)

	CreateOrder(order *models.OrderCore) (*models.OrderCore, error)
	GetOrderByID(id string) (*models.OrderCore, error)
	GetOrderByNumber(orderNumber string) (*models.OrderCore, error)
	GetOrderByPaymentID(paymentID string) (*models.OrderCore, error)
	UpdateOrder(order *models.OrderCore) error
	// WithOrderLock runs fn inside a transaction with SELECT FOR UPDATE on the order row.
	WithOrderLock(orderID string, fn func(order *models.OrderCore) error) error

	CreatePaymentAttempt(attempt *models.PaymentAttemptCore) (*models.PaymentAttemptCore, error)
}
