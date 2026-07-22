package delegate

import (
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
	"go.uber.org/fx"
)

type PaymentsDelegateImpl struct {
	usecase payments.UseCase
}

type PaymentsDelegateModule struct {
	fx.Out
	payments.Delegate
}

func SetupPaymentsDelegate(usecase payments.UseCase) PaymentsDelegateModule {
	return PaymentsDelegateModule{
		Delegate: &PaymentsDelegateImpl{usecase: usecase},
	}
}

func (d *PaymentsDelegateImpl) ListActiveProducts() ([]*models.ProductCore, error) {
	return d.usecase.ListActiveProducts()
}

func (d *PaymentsDelegateImpl) CreateProduct(input models.CreateProductInput) (*models.ProductCore, error) {
	return d.usecase.CreateProduct(input)
}

func (d *PaymentsDelegateImpl) Checkout(lmsUserID, productID, customerEmail string) (string, *models.OrderCore, error) {
	return d.usecase.Checkout(lmsUserID, productID, customerEmail)
}

func (d *PaymentsDelegateImpl) GetOrder(lmsUserID, orderNumber string) (*models.OrderCore, error) {
	return d.usecase.GetOrder(lmsUserID, orderNumber)
}

func (d *PaymentsDelegateImpl) SyncOrder(lmsUserID, orderNumber, sourceIP string) (*models.OrderCore, error) {
	return d.usecase.SyncOrder(lmsUserID, orderNumber, sourceIP)
}

func (d *PaymentsDelegateImpl) HandleWebhook(rawBody []byte, sourceIP string) error {
	return d.usecase.HandleWebhook(rawBody, sourceIP)
}
