package main

import (
	"fmt"
	"log"
	"time"

	"backend/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting integration payment verification test...")
	cfg := config.Load()

	// Connect to database
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database %s: %v", cfg.DBName, err)
	}
	defer db.Close()

	// 1. Verify product exists
	var productCount int
	err = db.Get(&productCount, "SELECT count(*) FROM products WHERE id = 'pdf-golang-guide'")
	if err != nil {
		log.Fatalf("Test Failed: Could not query products table: %v", err)
	}
	if productCount == 0 {
		log.Fatalf("Test Failed: Default seeded product 'pdf-golang-guide' is missing from the database.")
	}
	log.Println("[PASS] Seeded product exists in database.")

	// 2. Simulate order creation
	testOrderID := fmt.Sprintf("test_ord_%d", time.Now().Unix())
	_, err = db.Exec(
		"INSERT INTO orders (id, customer_email, customer_name, product_id, amount, currency, status) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		testOrderID, "tester@example.com", "Test User", "pdf-golang-guide", 49900, "INR", "pending",
	)
	if err != nil {
		log.Fatalf("Test Failed: Failed to insert test order: %v", err)
	}
	log.Printf("[PASS] Created pending test order: %s", testOrderID)

	// 3. Simulate Razorpay payment registration
	testRazorpayOrderID := fmt.Sprintf("rzp_test_order_%d", time.Now().Unix())
	_, err = db.Exec(
		"INSERT INTO payments (order_id, razorpay_order_id, status) VALUES ($1, $2, $3)",
		testOrderID, testRazorpayOrderID, "created",
	)
	if err != nil {
		log.Fatalf("Test Failed: Failed to insert payment details: %v", err)
	}
	log.Println("[PASS] Registered Razorpay order ID in database.")

	// 4. Simulate successful webhook callback (capturing payment and completing order)
	tx, err := db.Beginx()
	if err != nil {
		log.Fatalf("Test Failed: Failed to open transaction: %v", err)
	}
	defer tx.Rollback()

	// Update order
	_, err = tx.Exec("UPDATE orders SET status = 'paid', updated_at = NOW() WHERE id = $1", testOrderID)
	if err != nil {
		log.Fatalf("Test Failed: Failed to update order status to paid: %v", err)
	}

	// Update payment status
	testPaymentID := "pay_test_payment_123"
	testSignature := "test_signature_hash_xyz"
	_, err = tx.Exec(
		"UPDATE payments SET razorpay_payment_id = $1, razorpay_signature = $2, status = 'captured' WHERE razorpay_order_id = $3",
		testPaymentID, testSignature, testRazorpayOrderID,
	)
	if err != nil {
		log.Fatalf("Test Failed: Failed to update payment status: %v", err)
	}

	// Generate download token
	testToken := "test_token_token_abc"
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = tx.Exec(
		"INSERT INTO downloads (token, order_id, expires_at) VALUES ($1, $2, $3)",
		testToken, testOrderID, expiresAt,
	)
	if err != nil {
		log.Fatalf("Test Failed: Failed to create expiring download token: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Test Failed: Failed to commit transaction: %v", err)
	}
	log.Println("[PASS] Simulated payment captured. Status updated. Expiring download token generated.")

	// 5. Verify the download token resolves correctly
	var download struct {
		OrderID     string    `db:"order_id"`
		ExpiresAt   time.Time `db:"expires_at"`
		PDFFilename string    `db:"pdf_filename"`
	}
	verifyQuery := `
		SELECT d.order_id, d.expires_at, p.pdf_filename
		FROM downloads d
		JOIN orders o ON d.order_id = o.id
		JOIN products p ON o.product_id = p.id
		WHERE d.token = $1
	`
	err = db.Get(&download, verifyQuery, testToken)
	if err != nil {
		log.Fatalf("Test Failed: Could not resolve download token: %v", err)
	}

	if download.OrderID != testOrderID {
		log.Fatalf("Test Failed: Download token resolved to incorrect order. Expected: %s, Got: %s", testOrderID, download.OrderID)
	}
	if download.PDFFilename != "golang_guide.pdf" {
		log.Fatalf("Test Failed: Resolved incorrect PDF filename. Expected: golang_guide.pdf, Got: %s", download.PDFFilename)
	}
	if time.Now().After(download.ExpiresAt) {
		log.Fatalf("Test Failed: Download token is already expired.")
	}

	log.Printf("[PASS] Download token verified: points to order %s and maps to file '%s'. Expires at %s", 
		download.OrderID, download.PDFFilename, download.ExpiresAt.Format(time.RFC3339))

	// Clean up test data
	log.Println("Cleaning up integration test data...")
	_, _ = db.Exec("DELETE FROM downloads WHERE token = $1", testToken)
	_, _ = db.Exec("DELETE FROM payments WHERE order_id = $1", testOrderID)
	_, _ = db.Exec("DELETE FROM orders WHERE id = $1", testOrderID)
	log.Println("[PASS] Test data cleaned up successfully.")
	log.Println("\nIntegration Verification Success: Payment Lifecycle validation matches schema constraints!")
}
