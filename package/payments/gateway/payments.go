package gateway

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/db_client"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/payments"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaymentsGatewayImpl struct {
	db *gorm.DB
}

type PaymentsGatewayModule struct {
	fx.Out
	payments.Gateway
}

func SetupPaymentsGateway(postgresClient db_client.PostgresClient) PaymentsGatewayModule {
	_ = postgresClient
	dsn := viper.GetString("licensingPostgres.postgresDsn")
	if dsn == "" {
		panic("licensingPostgres.postgresDsn (or env LICENSING_POSTGRES_DSN) is required for payments")
	}
	db, err := db_client.OpenByDSN(dsn)
	if err != nil {
		panic(err)
	}
	return PaymentsGatewayModule{
		Gateway: &PaymentsGatewayImpl{db: db},
	}
}

func (g *PaymentsGatewayImpl) ListActiveProducts() ([]*models.ProductCore, error) {
	var rows []models.ProductDB
	if err := g.db.Where("is_active = ?", true).Order("amount ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*models.ProductCore, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToCore())
	}
	return out, nil
}

func (g *PaymentsGatewayImpl) GetProductByID(id string) (*models.ProductCore, error) {
	var row models.ProductDB
	err := g.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, payments.ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *PaymentsGatewayImpl) CreateProduct(product *models.ProductCore) (*models.ProductCore, error) {
	row := models.ProductDB{
		SKU:          product.SKU,
		Title:        product.Title,
		Description:  product.Description,
		Amount:       product.Amount,
		Currency:     product.Currency,
		SeatLimit:    product.SeatLimit,
		Capabilities: pq.StringArray(product.Capabilities),
		DurationDays: product.DurationDays,
		IsActive:     product.IsActive,
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *PaymentsGatewayImpl) CreateOrder(order *models.OrderCore) (*models.OrderCore, error) {
	row := models.OrderDB{
		OrderNumber:    order.OrderNumber,
		LmsUserID:      order.LmsUserID,
		ProductID:      order.ProductID,
		Amount:         order.Amount,
		Currency:       order.Currency,
		Status:         order.Status,
		IdempotencyKey: order.IdempotencyKey,
	}
	if order.YookassaPaymentID != "" {
		row.YookassaPaymentID = &order.YookassaPaymentID
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *PaymentsGatewayImpl) GetOrderByID(id string) (*models.OrderCore, error) {
	var row models.OrderDB
	err := g.db.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, payments.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *PaymentsGatewayImpl) GetOrderByNumber(orderNumber string) (*models.OrderCore, error) {
	var row models.OrderDB
	err := g.db.Where("order_number = ?", orderNumber).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, payments.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *PaymentsGatewayImpl) GetOrderByPaymentID(paymentID string) (*models.OrderCore, error) {
	var row models.OrderDB
	err := g.db.Where("yookassa_payment_id = ?", paymentID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, payments.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}

func (g *PaymentsGatewayImpl) UpdateOrder(order *models.OrderCore) error {
	updates := map[string]interface{}{
		"status":     order.Status,
		"updated_at": time.Now().UTC(),
	}
	if order.YookassaPaymentID != "" {
		updates["yookassa_payment_id"] = order.YookassaPaymentID
	}
	if order.LicenseID != "" {
		updates["license_id"] = order.LicenseID
	}
	if order.PaidAt != nil {
		updates["paid_at"] = *order.PaidAt
	}
	return g.db.Model(&models.OrderDB{}).Where("id = ?", order.ID).Updates(updates).Error
}

func (g *PaymentsGatewayImpl) WithOrderLock(orderID string, fn func(order *models.OrderCore) error) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		var row models.OrderDB
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return payments.ErrOrderNotFound
		}
		if err != nil {
			return err
		}
		core := row.ToCore()
		if err := fn(core); err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":     core.Status,
			"updated_at": time.Now().UTC(),
		}
		if core.YookassaPaymentID != "" {
			updates["yookassa_payment_id"] = core.YookassaPaymentID
		}
		if core.LicenseID != "" {
			updates["license_id"] = core.LicenseID
		}
		if core.PaidAt != nil {
			updates["paid_at"] = *core.PaidAt
		}
		return tx.Model(&models.OrderDB{}).Where("id = ?", orderID).Updates(updates).Error
	})
}

func (g *PaymentsGatewayImpl) CreatePaymentAttempt(attempt *models.PaymentAttemptCore) (*models.PaymentAttemptCore, error) {
	payload, err := json.Marshal(attempt.Payload)
	if err != nil {
		payload = []byte("{}")
	}
	row := models.PaymentAttemptDB{
		OrderID:   attempt.OrderID,
		EventType: attempt.EventType,
		Status:    attempt.Status,
		Payload:   payload,
	}
	if attempt.SourceIP != "" {
		row.SourceIP = &attempt.SourceIP
	}
	if err := g.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return row.ToCore(), nil
}
