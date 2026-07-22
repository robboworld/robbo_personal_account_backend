package payments

import "github.com/skinnykaen/robbo_student_personal_account.git/package/models"

// UseCase contains business logic for catalog, checkout and fulfillment.
type UseCase interface {
	ListActiveProducts() ([]*models.ProductCore, error)
	CreateProduct(input models.CreateProductInput) (*models.ProductCore, error)

	Checkout(lmsUserID, productID, customerEmail string) (confirmationURL string, order *models.OrderCore, err error)
	GetOrder(lmsUserID, orderNumber string) (*models.OrderCore, error)
	SyncOrder(lmsUserID, orderNumber, sourceIP string) (*models.OrderCore, error)

	HandleWebhook(rawBody []byte, sourceIP string) error
}
