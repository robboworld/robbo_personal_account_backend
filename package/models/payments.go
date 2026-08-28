package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

const (
	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusFailed   = "failed"
	OrderStatusCanceled = "canceled"
	OrderStatusRefunded = "refunded"

	ProductKindFixed       = "fixed"
	ProductKindConstructor = "constructor"

	PaymentAttemptReceived  = "received"
	PaymentAttemptSucceeded = "succeeded"
	PaymentAttemptCanceled  = "canceled"
	PaymentAttemptError     = "error"
)

// PricingConfigAxis describes min/max/step/included and unit price for one constructor axis.
type PricingConfigAxis struct {
	Included int     `json:"included"`
	Min      int     `json:"min"`
	Max      int     `json:"max"`
	Step     int     `json:"step"`
	ExtraRub float64 `json:"extraRub"`
}

// PricingConfigCloudAxis uses extra100MbRub for storage increments.
type PricingConfigCloudAxis struct {
	Included      int     `json:"included"`
	Min           int     `json:"min"`
	Max           int     `json:"max"`
	Step          int     `json:"step"`
	Extra100MbRub float64 `json:"extra100MbRub"`
}

// PricingConfig is stored in products.pricing_config for constructor tariffs.
type PricingConfig struct {
	BaseAmountRub float64                `json:"baseAmountRub"`
	Cloud         PricingConfigCloudAxis `json:"cloud"`
	Devices       PricingConfigAxis      `json:"devices"`
	Sessions      PricingConfigAxis      `json:"sessions"`
}

// ProductDB is the GORM model for products table.
type ProductDB struct {
	ID            string          `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	SKU           string          `gorm:"column:sku;uniqueIndex;not null"`
	Title         string          `gorm:"column:title;not null"`
	Description   string          `gorm:"column:description;not null;default:''"`
	Amount        float64         `gorm:"column:amount;type:numeric(12,2);not null"`
	Currency      string          `gorm:"column:currency;not null;default:RUB"`
	SeatLimit     int             `gorm:"column:seat_limit;not null;default:1"`
	Capabilities  pq.StringArray  `gorm:"column:capabilities;type:text[];not null"`
	CloudQuotaMB  int             `gorm:"column:cloud_quota_mb;not null;default:10"`
	SessionLimit  int             `gorm:"column:session_limit;not null;default:0"`
	DurationDays  int             `gorm:"column:duration_days;not null;default:365"`
	Kind          string          `gorm:"column:kind;not null;default:fixed"`
	PricingConfig json.RawMessage `gorm:"column:pricing_config;type:jsonb"`
	IsActive      bool            `gorm:"column:is_active;not null;default:true"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (ProductDB) TableName() string { return "products" }

// OrderDB is the GORM model for orders table.
type OrderDB struct {
	ID                string     `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderNumber       string     `gorm:"column:order_number;uniqueIndex;not null"`
	LmsUserID         string     `gorm:"column:lms_user_id;not null;index"`
	ProductID         string     `gorm:"column:product_id;type:uuid;not null"`
	Amount            float64    `gorm:"column:amount;type:numeric(12,2);not null"`
	Currency          string     `gorm:"column:currency;not null;default:RUB"`
	Status            string     `gorm:"column:status;not null;default:pending"`
	IdempotencyKey    string     `gorm:"column:idempotency_key;uniqueIndex;not null"`
	YookassaPaymentID *string    `gorm:"column:yookassa_payment_id;uniqueIndex"`
	LicenseID         *string    `gorm:"column:license_id;type:uuid"`
	CloudQuotaMB      *int       `gorm:"column:cloud_quota_mb"`
	SeatLimit         *int       `gorm:"column:seat_limit"`
	SessionLimit      *int       `gorm:"column:session_limit"`
	PaidAt            *time.Time `gorm:"column:paid_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (OrderDB) TableName() string { return "orders" }

// PaymentAttemptDB is the GORM model for payment_attempts table.
type PaymentAttemptDB struct {
	ID        string          `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID   string          `gorm:"column:order_id;type:uuid;not null;index"`
	EventType string          `gorm:"column:event_type;not null;default:''"`
	Status    string          `gorm:"column:status;not null;default:received"`
	SourceIP  *string         `gorm:"column:source_ip"`
	Payload   json.RawMessage `gorm:"column:payload;type:jsonb;not null"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (PaymentAttemptDB) TableName() string { return "payment_attempts" }

// ProductCore is the domain product model.
type ProductCore struct {
	ID            string
	SKU           string
	Title         string
	Description   string
	Amount        float64
	Currency      string
	SeatLimit     int
	Capabilities  []string
	CloudQuotaMB  int
	SessionLimit  int
	DurationDays  int
	Kind          string
	PricingConfig *PricingConfig
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (p *ProductCore) IsConstructor() bool {
	return p != nil && p.Kind == ProductKindConstructor
}

// OrderCore is the domain order model.
type OrderCore struct {
	ID                string
	OrderNumber       string
	LmsUserID         string
	ProductID         string
	Amount            float64
	Currency          string
	Status            string
	IdempotencyKey    string
	YookassaPaymentID string
	LicenseID         string
	CloudQuotaMB      int
	SeatLimit         int
	SessionLimit      int
	PaidAt            *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// HasLimitSnapshot is true when the order stores constructor limits (non-zero or explicitly set).
func (o *OrderCore) HasLimitSnapshot() bool {
	return o != nil && (o.CloudQuotaMB > 0 || o.SeatLimit > 0 || o.SessionLimit > 0)
}

// PaymentAttemptCore is the domain payment attempt model.
type PaymentAttemptCore struct {
	ID        string
	OrderID   string
	EventType string
	Status    string
	SourceIP  string
	Payload   map[string]interface{}
	CreatedAt time.Time
}

// CreateProductInput is used by SuperAdmin to seed catalog entries.
type CreateProductInput struct {
	SKU          string
	Title        string
	Description  string
	Amount       float64
	Currency     string
	SeatLimit    int
	Capabilities []string
	CloudQuotaMB int
	SessionLimit int
	DurationDays int
	IsActive     bool
}

func (db *ProductDB) ToCore() *ProductCore {
	caps := make([]string, len(db.Capabilities))
	copy(caps, db.Capabilities)
	var pricing *PricingConfig
	if len(db.PricingConfig) > 0 && string(db.PricingConfig) != "null" {
		_ = json.Unmarshal(db.PricingConfig, &pricing)
	}
	kind := db.Kind
	if kind == "" {
		kind = ProductKindFixed
	}
	return &ProductCore{
		ID:            db.ID,
		SKU:           db.SKU,
		Title:         db.Title,
		Description:   db.Description,
		Amount:        db.Amount,
		Currency:      db.Currency,
		SeatLimit:     db.SeatLimit,
		Capabilities:  caps,
		CloudQuotaMB:  db.CloudQuotaMB,
		SessionLimit:  db.SessionLimit,
		DurationDays:  db.DurationDays,
		Kind:          kind,
		PricingConfig: pricing,
		IsActive:      db.IsActive,
		CreatedAt:     db.CreatedAt,
		UpdatedAt:     db.UpdatedAt,
	}
}

func (db *OrderDB) ToCore() *OrderCore {
	core := &OrderCore{
		ID:             db.ID,
		OrderNumber:    db.OrderNumber,
		LmsUserID:      db.LmsUserID,
		ProductID:      db.ProductID,
		Amount:         db.Amount,
		Currency:       db.Currency,
		Status:         db.Status,
		IdempotencyKey: db.IdempotencyKey,
		PaidAt:         db.PaidAt,
		CreatedAt:      db.CreatedAt,
		UpdatedAt:      db.UpdatedAt,
	}
	if db.YookassaPaymentID != nil {
		core.YookassaPaymentID = *db.YookassaPaymentID
	}
	if db.LicenseID != nil {
		core.LicenseID = *db.LicenseID
	}
	if db.CloudQuotaMB != nil {
		core.CloudQuotaMB = *db.CloudQuotaMB
	}
	if db.SeatLimit != nil {
		core.SeatLimit = *db.SeatLimit
	}
	if db.SessionLimit != nil {
		core.SessionLimit = *db.SessionLimit
	}
	return core
}

func (db *PaymentAttemptDB) ToCore() *PaymentAttemptCore {
	payload := map[string]interface{}{}
	if len(db.Payload) > 0 {
		_ = json.Unmarshal(db.Payload, &payload)
	}
	core := &PaymentAttemptCore{
		ID:        db.ID,
		OrderID:   db.OrderID,
		EventType: db.EventType,
		Status:    db.Status,
		Payload:   payload,
		CreatedAt: db.CreatedAt,
	}
	if db.SourceIP != nil {
		core.SourceIP = *db.SourceIP
	}
	return core
}
