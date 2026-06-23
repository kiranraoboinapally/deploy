package main

import (
	"fmt"
	"log"

	"backend/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting database migrations...")
	cfg := config.Load()

	// 1. Connect to default 'postgres' database to ensure the target DB exists
	postgresDsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBSSLMode,
	)

	db, err := sqlx.Connect("postgres", postgresDsn)
	if err != nil {
		log.Fatalf("Failed to connect to default postgres database: %v", err)
	}

	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", cfg.DBName)
	err = db.Get(&exists, query)
	if err != nil {
		db.Close()
		log.Fatalf("Failed to check if database exists: %v", err)
	}

	if !exists {
		log.Printf("Database %s does not exist. Creating it...", cfg.DBName)
		// CREATE DATABASE cannot be executed inside a transaction block.
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
		if err != nil {
			db.Close()
			log.Fatalf("Failed to create database %s: %v", cfg.DBName, err)
		}
		log.Printf("Database %s created successfully.", cfg.DBName)
	} else {
		log.Printf("Database %s already exists.", cfg.DBName)
	}
	db.Close()

	// 2. Connect to the newly created / existing target database
	targetDsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	targetDB, err := sqlx.Connect("postgres", targetDsn)
	if err != nil {
		log.Fatalf("Failed to connect to target database %s: %v", cfg.DBName, err)
	}
	defer targetDB.Close()

	// 3. Create tables
	log.Println("Creating tables if they do not exist...")

	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id VARCHAR(100) PRIMARY KEY,
		squarespace_id VARCHAR(100),
		name VARCHAR(255) NOT NULL,
		price INT NOT NULL, -- in paise
		currency VARCHAR(10) DEFAULT 'INR',
		pdf_filename VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS orders (
		id VARCHAR(100) PRIMARY KEY,
		customer_email VARCHAR(255) NOT NULL,
		customer_name VARCHAR(255) NOT NULL,
		product_id VARCHAR(100) REFERENCES products(id),
		amount INT NOT NULL, -- in paise
		currency VARCHAR(10) DEFAULT 'INR',
		status VARCHAR(50) DEFAULT 'pending',
		squarespace_order_id VARCHAR(100),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS payments (
		id SERIAL PRIMARY KEY,
		order_id VARCHAR(100) REFERENCES orders(id),
		razorpay_order_id VARCHAR(255) UNIQUE NOT NULL,
		razorpay_payment_id VARCHAR(255),
		razorpay_signature VARCHAR(255),
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS downloads (
		token VARCHAR(255) PRIMARY KEY,
		order_id VARCHAR(100) REFERENCES orders(id),
		expires_at TIMESTAMP NOT NULL,
		download_count INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = targetDB.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to execute schema creation: %v", err)
	}
	log.Println("Tables created successfully.")

	// 4. Seed default PDF product
	log.Println("Seeding default product...")
	seedProduct := `
	INSERT INTO products (id, squarespace_id, name, price, currency, pdf_filename)
	VALUES ('pdf-golang-guide', 'squarespace-pdf-variant-1', 'Mastering Go: A Complete Guide (PDF)', 49900, 'INR', 'golang_guide.pdf')
	ON CONFLICT (id) DO NOTHING;
	`
	_, err = targetDB.Exec(seedProduct)
	if err != nil {
		log.Fatalf("Failed to seed default product: %v", err)
	}

	// Verify seeding
	var count int
	err = targetDB.Get(&count, "SELECT count(*) FROM products")
	if err != nil {
		log.Printf("Failed to verify product seeding: %v", err)
	} else {
		log.Printf("Database initialized. Total products in database: %d", count)
	}
	log.Println("Migrations completed successfully!")
}
