package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend/config"
)

type SquarespaceService struct {
	cfg *config.Config
}

type MonetaryAmount struct {
	Currency string `json:"currency"`
	Value    string `json:"value"` // e.g. "499.00"
}

type Address struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Address1    string `json:"address1"`
	City        string `json:"city"`
	State       string `json:"state"`
	CountryCode string `json:"countryCode"`
	PostalCode  string `json:"postalCode"`
}

type SquarespaceLineItem struct {
	ProductId     string         `json:"productId,omitempty"`
	VariantId     string         `json:"variantId,omitempty"`
	Quantity      int            `json:"quantity"`
	UnitPricePaid MonetaryAmount `json:"unitPricePaid"`
}

type SquarespaceOrderPayload struct {
	ChannelName            string                `json:"channelName"`
	CreatedOn              string                `json:"createdOn"` // ISO 8601
	ExternalOrderReference string                `json:"externalOrderReference"`
	CustomerEmail          string                `json:"customerEmail"`
	BillingAddress         Address               `json:"billingAddress"`
	ShippingAddress        Address               `json:"shippingAddress"`
	LineItems              []SquarespaceLineItem `json:"lineItems"`
	Subtotal               MonetaryAmount        `json:"subtotal"`
	TaxTotal               MonetaryAmount        `json:"taxTotal"`
	ShippingTotal          MonetaryAmount        `json:"shippingTotal"`
	TestMode               bool                  `json:"testmode"`
}

type SquarespaceOrderResponse struct {
	ID string `json:"id"`
	// Additional fields omitted for simplicity
}

func NewSquarespaceService(cfg *config.Config) *SquarespaceService {
	return &SquarespaceService{cfg: cfg}
}

// SyncOrder imports an externally paid order into Squarespace
func (s *SquarespaceService) SyncOrder(orderID string, email string, name string, productID string, squarespaceVariantID string, productName string, amountPaise int, currency string) (string, error) {
	if s.cfg.SquarespaceAPIKey == "" {
		log.Printf("[Squarespace Service] API key not configured. Mocking order sync for order: %s", orderID)
		return "mock-sq-order-" + orderID, nil
	}

	url := "https://api.squarespace.com/1.0/commerce/orders"

	// Format price from paise to standard decimal representation, e.g. 49900 -> 499.00
	amountDecimal := fmt.Sprintf("%.2f", float64(amountPaise)/100.0)

	// Set up simple placeholder address since PDF product doesn't require physical shipping
	address := Address{
		FirstName:   name,
		LastName:    "Customer",
		Address1:    "Digital Delivery",
		City:        "Internet",
		State:       "IN",
		CountryCode: "IN",
		PostalCode:  "000000",
	}

	payload := SquarespaceOrderPayload{
		ChannelName:            "Razorpay Checkout",
		CreatedOn:              time.Now().UTC().Format(time.RFC3339),
		ExternalOrderReference: orderID,
		CustomerEmail:          email,
		BillingAddress:         address,
		ShippingAddress:        address,
		LineItems: []SquarespaceLineItem{
			{
				ProductId: productID, // Map our local product ID or the Squarespace Product ID
				VariantId: squarespaceVariantID,
				Quantity:  1,
				UnitPricePaid: MonetaryAmount{
					Currency: currency,
					Value:    amountDecimal,
				},
			},
		},
		Subtotal: MonetaryAmount{
			Currency: currency,
			Value:    amountDecimal,
		},
		TaxTotal: MonetaryAmount{
			Currency: currency,
			Value:    "0.00",
		},
		ShippingTotal: MonetaryAmount{
			Currency: currency,
			Value:    "0.00",
		},
		TestMode: true, // Mark true for sandboxed imports
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.SquarespaceAPIKey)
	req.Header.Set("Idempotency-Key", orderID)
	req.Header.Set("User-Agent", "Razorpay-Squarespace-Custom-Checkout/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errData map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errData)
		return "", fmt.Errorf("squarespace returned error status %d: %v", resp.StatusCode, errData)
	}

	var sqResponse SquarespaceOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&sqResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return sqResponse.ID, nil
}
