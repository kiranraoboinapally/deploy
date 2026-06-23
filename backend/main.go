package main

import (
	"encoding/json"
	"log"
	"net/http"

	"backend/config"
	"backend/database"
	"backend/server"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize the payment server
	paymentSrv := server.NewPaymentServer(db, cfg)

	// Health check endpoint
	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	// Register payment integration APIs with CORS middleware enabled
	http.HandleFunc("/api/products", paymentSrv.EnableCors(paymentSrv.GetProducts))
	http.HandleFunc("/api/checkout/create-order", paymentSrv.EnableCors(paymentSrv.CreateOrder))
	http.HandleFunc("/api/checkout/verify-payment", paymentSrv.EnableCors(paymentSrv.VerifyPayment))
	
	// Trailing slash enables matching paths like /api/download/<token>
	http.HandleFunc("/api/download/", paymentSrv.EnableCors(paymentSrv.DownloadFile))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
