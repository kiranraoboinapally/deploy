package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend/config"
)

type RazorpayService struct {
	cfg *config.Config
}

type RazorpayOrderResponse struct {
	ID       string `json: "id"`
	Entity   string `json: "entity"`
	Amount   int    `json: "amount"`
	Currency string `json: "currency"`
	Receipt  string `json: "receipt"`
	Status   string `json: "status"`
}

func NewRazorpayService(cfg *config.Config) *RazorpayService {
	return &RazorpayService{cfg: cfg}
}

// CreateOrder creates a Razorpay order
func (s *RazorpayService) CreateOrder(orderID string, amount int, currency string) (*RazorpayOrderResponse, error) {
	if s.cfg.RazorpayKeyID == "" || s.cfg.RazorpayKeySecret == "" {
		return nil, fmt.Errorf("Razorpay credentials are not configured in environment variables")
	}

	url := "https://api.razorpay.com/v1/orders"
	payload := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  orderID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.cfg.RazorpayKeyID, s.cfg.RazorpayKeySecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errData map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errData)
		return nil, fmt.Errorf("razorpay returned error status %d: %v", resp.StatusCode, errData)
	}

	var razorpayOrder RazorpayOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&razorpayOrder); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &razorpayOrder, nil
}

// VerifySignature verifies the signature returned by Razorpay Checkout
func (s *RazorpayService) VerifySignature(razorpayOrderID, razorpayPaymentID, signature string) bool {
	if s.cfg.RazorpayKeySecret == "" {
		return false
	}

	data := razorpayOrderID + "|" + razorpayPaymentID
	h := hmac.New(sha256.New, []byte(s.cfg.RazorpayKeySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
