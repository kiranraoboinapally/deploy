package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"backend/config"
	"backend/services"

	"github.com/jmoiron/sqlx"
)

type PaymentServer struct {
	DB          *sqlx.DB
	Cfg         *config.Config
	Razorpay    *services.RazorpayService
	Squarespace *services.SquarespaceService
	Email       *services.EmailService
}

func NewPaymentServer(db *sqlx.DB, cfg *config.Config) *PaymentServer {
	return &PaymentServer{
		DB:          db,
		Cfg:         cfg,
		Razorpay:    services.NewRazorpayService(cfg),
		Squarespace: services.NewSquarespaceService(cfg),
		Email:       services.NewEmailService(cfg),
	}
}

// Enable CORS middleware
func (s *PaymentServer) EnableCors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// Generate secure random download token
func generateSecureToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// Fallback timestamp if crypto rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// GET /api/products
func (s *PaymentServer) GetProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type Product struct {
		ID             string `db:"id" json:"id"`
		SquarespaceID  string `db:"squarespace_id" json:"squarespace_id"`
		Name           string `db:"name" json:"name"`
		Price          int    `db:"price" json:"price"`
		Currency       string `db:"currency" json:"currency"`
		PDFFilename    string `db:"pdf_filename" json:"pdf_filename"`
	}

	var products []Product
	err := s.DB.Select(&products, "SELECT id, squarespace_id, name, price, currency, pdf_filename FROM products")
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// POST /api/checkout/create-order
func (s *PaymentServer) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProductID string `json:"product_id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.ProductID == "" || req.Email == "" || req.Name == "" {
		http.Error(w, "product_id, email, and name are required", http.StatusBadRequest)
		return
	}

	// Fetch product details
	var product struct {
		ID       string `db:"id"`
		Name     string `db:"name"`
		Price    int    `db:"price"`
		Currency string `db:"currency"`
	}
	err := s.DB.Get(&product, "SELECT id, name, price, currency FROM products WHERE id = $1", req.ProductID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Product not found", http.StatusNotFound)
		} else {
			log.Printf("Error fetching product: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Generate local order ID
	localOrderID := fmt.Sprintf("ord_%d_%s", time.Now().Unix(), generateSecureToken()[:8])

	// Create Razorpay Order
	rzpOrder, err := s.Razorpay.CreateOrder(localOrderID, product.Price, product.Currency)
	if err != nil {
		log.Printf("Error creating Razorpay order: %v", err)
		http.Error(w, "Payment provider integration error", http.StatusBadGateway)
		return
	}

	// Save to DB inside transaction
	tx, err := s.DB.Beginx()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		http.Error(w, "Database transaction error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert order
	_, err = tx.Exec(
		"INSERT INTO orders (id, customer_email, customer_name, product_id, amount, currency, status) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		localOrderID, req.Email, req.Name, product.ID, product.Price, product.Currency, "pending",
	)
	if err != nil {
		log.Printf("Failed to insert order: %v", err)
		http.Error(w, "Database insert error", http.StatusInternalServerError)
		return
	}

	// Insert payment
	_, err = tx.Exec(
		"INSERT INTO payments (order_id, razorpay_order_id, status) VALUES ($1, $2, $3)",
		localOrderID, rzpOrder.ID, "created",
	)
	if err != nil {
		log.Printf("Failed to insert payment: %v", err)
		http.Error(w, "Database insert error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		http.Error(w, "Database commit error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"order_id":          localOrderID,
		"razorpay_order_id": rzpOrder.ID,
		"amount":            product.Price,
		"currency":          product.Currency,
		"razorpay_key_id":   s.Cfg.RazorpayKeyID,
		"product_name":      product.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// POST /api/checkout/verify-payment
func (s *PaymentServer) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RazorpayOrderID   string `json:"razorpay_order_id"`
		RazorpayPaymentID string `json:"razorpay_payment_id"`
		RazorpaySignature string `json:"razorpay_signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.RazorpayOrderID == "" || req.RazorpayPaymentID == "" || req.RazorpaySignature == "" {
		http.Error(w, "razorpay_order_id, razorpay_payment_id, and razorpay_signature are required", http.StatusBadRequest)
		return
	}

	// 1. Verify Razorpay signature
	verified := s.Razorpay.VerifySignature(req.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature)
	if !verified {
		log.Printf("Razorpay payment signature mismatch for order %s", req.RazorpayOrderID)
		http.Error(w, "Invalid payment signature verification failed", http.StatusBadRequest)
		return
	}

	// 2. Fetch order associated with this Razorpay order
	var dbOrder struct {
		ID            string `db:"order_id"`
		LocalOrderID  string `db:"id"`
		CustomerEmail string `db:"customer_email"`
		CustomerName  string `db:"customer_name"`
		ProductID     string `db:"product_id"`
		ProductName   string `db:"product_name"`
		SquarespaceID string `db:"squarespace_id"`
		Amount        int    `db:"amount"`
		Currency      string `db:"currency"`
		Status        string `db:"status"`
	}

	query := `
		SELECT o.id, o.customer_email, o.customer_name, o.product_id, p.name AS product_name, p.squarespace_id, o.amount, o.currency, o.status, pay.order_id
		FROM payments pay
		JOIN orders o ON pay.order_id = o.id
		JOIN products p ON o.product_id = p.id
		WHERE pay.razorpay_order_id = $1
	`
	err := s.DB.Get(&dbOrder, query, req.RazorpayOrderID)
	if err != nil {
		log.Printf("Payment record not found in database: %v", err)
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Double check if payment is already processed
	if dbOrder.Status == "paid" {
		log.Printf("Order %s already processed. Returning existing tokens.", dbOrder.LocalOrderID)
		var existingToken string
		err = s.DB.Get(&existingToken, "SELECT token FROM downloads WHERE order_id = $1 LIMIT 1", dbOrder.LocalOrderID)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "success",
				"token":  existingToken,
			})
			return
		}
	}

	// 3. Mark transaction as paid inside a transaction
	tx, err := s.DB.Beginx()
	if err != nil {
		log.Printf("Failed to begin database transaction: %v", err)
		http.Error(w, "Database transaction error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE payments SET razorpay_payment_id = $1, razorpay_signature = $2, status = $3 WHERE razorpay_order_id = $4",
		req.RazorpayPaymentID, req.RazorpaySignature, "captured", req.RazorpayOrderID,
	)
	if err != nil {
		log.Printf("Failed to update payment details: %v", err)
		http.Error(w, "Database update error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2", "paid", dbOrder.LocalOrderID)
	if err != nil {
		log.Printf("Failed to update order status: %v", err)
		http.Error(w, "Database update error", http.StatusInternalServerError)
		return
	}

	// Generate secure expiring token
	downloadToken := generateSecureToken()
	expiresAt := time.Now().Add(24 * time.Hour) // Valid for 24 hours

	_, err = tx.Exec(
		"INSERT INTO downloads (token, order_id, expires_at) VALUES ($1, $2, $3)",
		downloadToken, dbOrder.LocalOrderID, expiresAt,
	)
	if err != nil {
		log.Printf("Failed to insert secure download token: %v", err)
		http.Error(w, "Database insert error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit db changes: %v", err)
		http.Error(w, "Database transaction commit error", http.StatusInternalServerError)
		return
	}

	// Get Host dynamically from Request for constructing the links
	backendHost := fmt.Sprintf("http://%s", r.Host)
	if r.Header.Get("X-Forwarded-Proto") != "" {
		backendHost = fmt.Sprintf("%s://%s", r.Header.Get("X-Forwarded-Proto"), r.Host)
	}

	// 4. Asynchronously Sync with Squarespace Orders API
	go func() {
		sqOrderID, err := s.Squarespace.SyncOrder(
			dbOrder.LocalOrderID,
			dbOrder.CustomerEmail,
			dbOrder.CustomerName,
			dbOrder.ProductID,
			dbOrder.SquarespaceID,
			dbOrder.ProductName,
			dbOrder.Amount,
			dbOrder.Currency,
		)
		if err != nil {
			log.Printf("[Sync Error] Failed to sync order %s to Squarespace: %v", dbOrder.LocalOrderID, err)
			return
		}
		log.Printf("[Sync Success] Order %s synced to Squarespace. SQ Order ID: %s", dbOrder.LocalOrderID, sqOrderID)

		// Save Squarespace Order ID
		_, err = s.DB.Exec("UPDATE orders SET squarespace_order_id = $1 WHERE id = $2", sqOrderID, dbOrder.LocalOrderID)
		if err != nil {
			log.Printf("Failed to store Squarespace order ID in DB: %v", err)
		}
	}()

	// 5. Send secure email
	err = s.Email.SendDownloadEmail(
		dbOrder.CustomerEmail,
		dbOrder.CustomerName,
		dbOrder.ProductName,
		downloadToken,
		backendHost,
	)
	if err != nil {
		log.Printf("[Email Error] Failed to send email to %s: %v", dbOrder.CustomerEmail, err)
		// We do not fail the request if email fails, because the customer has already paid.
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"token":  downloadToken,
	})
}

// GET /api/download/:token (Secure file server)
func (s *PaymentServer) DownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from URL path. e.g. path "/api/download/abcdef123" -> "abcdef123"
	token := r.URL.Path[len("/api/download/"):]
	if token == "" {
		token = r.URL.Query().Get("token") // Fallback
	}

	if token == "" {
		http.Error(w, "Download token is required", http.StatusBadRequest)
		return
	}

	// Query token validity and check product pdf name
	var download struct {
		Token         string    `db:"token"`
		OrderID       string    `db:"order_id"`
		ExpiresAt     time.Time `db:"expires_at"`
		DownloadCount int       `db:"download_count"`
		PDFFilename   string    `db:"pdf_filename"`
		ProductName   string    `db:"product_name"`
	}

	query := `
		SELECT d.token, d.order_id, d.expires_at, d.download_count, p.pdf_filename, p.name AS product_name
		FROM downloads d
		JOIN orders o ON d.order_id = o.id
		JOIN products p ON o.product_id = p.id
		WHERE d.token = $1
	`
	err := s.DB.Get(&download, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid or expired download link", http.StatusNotFound)
		} else {
			log.Printf("Database query error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Check expiration
	if time.Now().After(download.ExpiresAt) {
		http.Error(w, "This download link has expired. Secure links are valid for 24 hours only.", http.StatusGone)
		return
	}

	// Increment download count
	_, err = s.DB.Exec("UPDATE downloads SET download_count = download_count + 1 WHERE token = $1", token)
	if err != nil {
		log.Printf("Failed to increment download count for token %s: %v", token, err)
	}

	// Resolve file path
	storagePath := s.Cfg.PDFStoragePath
	if storagePath == "" {
		storagePath = "./storage/pdfs"
	}
	
	filePath := filepath.Join(storagePath, download.PDFFilename)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Requested PDF file not found at path %s", filePath)
		http.Error(w, "PDF file is currently unavailable on the server.", http.StatusNotFound)
		return
	}

	// Serve the file
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", download.PDFFilename))
	http.ServeFile(w, r, filePath)
}
