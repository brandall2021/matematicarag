package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database")

	users := []struct {
		email    string
		password string
		name     string
		lastName string
		role     string
	}{
		{"admin@face-unt.ar", "Admin123!", "Admin", "FACE", "ADMIN"},
		{"profesor@face-unt.ar", "Profesor123!", "Carlos", "Garcia", "TEACHER"},
		{"alumno@face-unt.ar", "Alumno123!", "Maria", "Lopez", "STUDENT"},
	}

	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password for %s: %v", u.email, err)
			continue
		}

		_, err = db.Exec(context.Background(),
			`INSERT INTO users (email, password_hash, name, last_name, role)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (email) DO NOTHING`,
			u.email, string(hash), u.name, u.lastName, u.role,
		)
		if err != nil {
			log.Printf("Error creating user %s: %v", u.email, err)
			continue
		}
		fmt.Printf("  Created: %s (%s)\n", u.email, u.role)
	}

	fmt.Println("")
	fmt.Println("=== Example Users ===")
	fmt.Println("Admin:    admin@face-unt.ar    / Admin123!")
	fmt.Println("Teacher:  profesor@face-unt.ar / Profesor123!")
	fmt.Println("Student:  alumno@face-unt.ar   / Alumno123!")
}
