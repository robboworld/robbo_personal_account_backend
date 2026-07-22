package yookassa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const apiBase = "https://api.yookassa.ru/v3"

// Trusted YooKassa webhook IP ranges (from YooKassa docs).
var trustedCIDRs = []string{
	"185.71.76.0/27",
	"185.71.77.0/27",
	"77.75.153.0/25",
	"77.75.156.11/32",
	"77.75.156.35/32",
	"2a02:5180::/32",
}

// Receipt is a 54-FZ receipt payload for YooKassa.
type Receipt struct {
	Customer *ReceiptCustomer `json:"customer,omitempty"`
	Items    []ReceiptItem    `json:"items"`
}

type ReceiptCustomer struct {
	Email string `json:"email,omitempty"`
}

type ReceiptItem struct {
	Description    string        `json:"description"`
	Quantity       string        `json:"quantity"`
	Amount         AmountValue   `json:"amount"`
	VatCode        int           `json:"vat_code"`
	PaymentMode    string        `json:"payment_mode"`
	PaymentSubject string        `json:"payment_subject"`
}

type AmountValue struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type CreatePaymentRequest struct {
	IdempotencyKey string
	Amount         float64
	Currency       string
	Description    string
	ReturnURL      string
	Metadata       map[string]string
	Receipt        *Receipt
}

type CreatePaymentResult struct {
	PaymentID       string
	ConfirmationURL string
	Status          string
}

type PaymentInfo struct {
	ID       string
	Status   string
	Metadata map[string]string
}

type Client struct {
	shopID     string
	secretKey  string
	httpClient *http.Client
}

func NewClientFromViper() *Client {
	return &Client{
		shopID:    strings.TrimSpace(viper.GetString("payments.yookassaShopId")),
		secretKey: strings.TrimSpace(viper.GetString("payments.yookassaSecretKey")),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) IsConfigured() bool {
	return c.shopID != "" && c.secretKey != ""
}

func BuildReceipt(amount float64, currency, description, customerEmail string) *Receipt {
	if !viper.GetBool("payments.sendReceipt") {
		return nil
	}
	vatCode := viper.GetInt("payments.receiptVatCode")
	if vatCode <= 0 {
		vatCode = 1
	}
	subject := viper.GetString("payments.receiptPaymentSubject")
	if subject == "" {
		subject = "service"
	}
	mode := viper.GetString("payments.receiptPaymentMode")
	if mode == "" {
		mode = "full_payment"
	}
	desc := description
	if len(desc) > 128 {
		desc = desc[:128]
	}
	receipt := &Receipt{
		Items: []ReceiptItem{
			{
				Description: desc,
				Quantity:    "1.00",
				Amount: AmountValue{
					Value:    formatAmount(amount),
					Currency: currency,
				},
				VatCode:        vatCode,
				PaymentMode:    mode,
				PaymentSubject: subject,
			},
		},
	}
	if customerEmail != "" {
		receipt.Customer = &ReceiptCustomer{Email: customerEmail}
	}
	return receipt
}

func (c *Client) CreatePayment(req CreatePaymentRequest) (*CreatePaymentResult, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("yookassa not configured")
	}
	payload := map[string]interface{}{
		"amount": map[string]string{
			"value":    formatAmount(req.Amount),
			"currency": req.Currency,
		},
		"confirmation": map[string]string{
			"type":       "redirect",
			"return_url": req.ReturnURL,
		},
		"capture":     true,
		"description": truncate(req.Description, 128),
		"metadata":    req.Metadata,
	}
	if req.Receipt != nil {
		payload["receipt"] = req.Receipt
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, apiBase+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotence-Key", req.IdempotencyKey)
	httpReq.SetBasicAuth(c.shopID, c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yookassa create payment status=%d body=%s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		Confirmation struct {
			ConfirmationURL string `json:"confirmation_url"`
		} `json:"confirmation"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if parsed.ID == "" || parsed.Confirmation.ConfirmationURL == "" {
		return nil, fmt.Errorf("yookassa create payment missing id/url")
	}
	return &CreatePaymentResult{
		PaymentID:       parsed.ID,
		ConfirmationURL: parsed.Confirmation.ConfirmationURL,
		Status:          parsed.Status,
	}, nil
}

func (c *Client) GetPayment(paymentID string) (*PaymentInfo, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("yookassa not configured")
	}
	httpReq, err := http.NewRequest(http.MethodGet, apiBase+"/payments/"+paymentID, nil)
	if err != nil {
		return nil, err
	}
	httpReq.SetBasicAuth(c.shopID, c.secretKey)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yookassa get payment status=%d body=%s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		ID       string            `json:"id"`
		Status   string            `json:"status"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return &PaymentInfo{
		ID:       parsed.ID,
		Status:   parsed.Status,
		Metadata: parsed.Metadata,
	}, nil
}

// ParseWebhookNotification extracts event type, payment id and metadata from webhook JSON.
func ParseWebhookNotification(body []byte) (event, paymentID string, metadata map[string]string, err error) {
	var parsed struct {
		Event  string `json:"event"`
		Object struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
		} `json:"object"`
	}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return "", "", nil, err
	}
	if parsed.Event == "" || parsed.Object.ID == "" {
		return "", "", nil, fmt.Errorf("invalid webhook payload")
	}
	return parsed.Event, parsed.Object.ID, parsed.Object.Metadata, nil
}

func IsTrustedIP(ipAddress string) bool {
	if ipAddress == "" {
		return false
	}
	ip := net.ParseIP(strings.TrimSpace(ipAddress))
	if ip == nil {
		return false
	}
	for _, cidr := range trustedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func formatAmount(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
