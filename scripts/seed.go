package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/telemed_hub?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// 1. Seed Specialties
	specialties := []struct {
		Name        string
		Icon        string
		Description string
	}{
		{"General Practitioner", "icon-gp.png", "Handles general health issues and initial diagnoses."},
		{"Cardiologist", "icon-cardiology.png", "Specializes in diagnosing and treating diseases of the cardiovascular system."},
		{"Pediatrician", "icon-pediatrics.png", "Focuses on the physical, emotional, and social health of children."},
		{"Dermatologist", "icon-dermatology.png", "Specializes in conditions involving the skin, hair, and nails."},
		{"Neurologist", "icon-neurology.png", "Treats disorders that affect the brain, spinal cord, and nerves."},
	}

	specialtyIDs := make(map[string]uuid.UUID)

	for _, spec := range specialties {
		id := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO specialties (id, name, image_icon, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING
		`, id, spec.Name, spec.Icon, spec.Description)
		
		if err != nil {
			log.Fatalf("Failed to insert specialty %s: %v", spec.Name, err)
		}

		// Fetch the ID in case it already existed
		var existingID uuid.UUID
		err = pool.QueryRow(ctx, "SELECT id FROM specialties WHERE name = $1", spec.Name).Scan(&existingID)
		if err == nil {
			specialtyIDs[spec.Name] = existingID
		} else {
			specialtyIDs[spec.Name] = id
		}
	}
	fmt.Println("Specialties seeded successfully!")

	// 2. Seed Doctors
	password := "Rahasia123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	doctors := []struct {
		FullName      string
		Email         string
		SpecialtyName string
		License       string
		Phone         string
		Fee           int
	}{
		{"Dr. Andi Pratama", "andi@telemed.com", "General Practitioner", "ID-GP-001", "+628111111111", 50000},
		{"Dr. Budi Santoso", "budi@telemed.com", "Cardiologist", "ID-CARD-002", "+628222222222", 150000},
		{"Dr. Citra Lestari", "citra@telemed.com", "Pediatrician", "ID-PED-003", "+628333333333", 100000},
		{"Dr. Dina Wijaya", "dina@telemed.com", "Dermatologist", "ID-DERM-004", "+628444444444", 120000},
		{"Dr. Eko Prasetyo", "eko@telemed.com", "Neurologist", "ID-NEURO-005", "+628555555555", 200000},
	}

	for _, doc := range doctors {
		userID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, full_name, is_verified, status)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (email) DO NOTHING
		`, userID, doc.Email, hashedPassword, doc.FullName, true, "active")
		
		if err != nil {
			log.Fatalf("Failed to insert user %s: %v", doc.Email, err)
		}

		// Get user ID if existed
		err = pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", doc.Email).Scan(&userID)
		if err != nil {
			log.Fatalf("Failed to retrieve user ID for %s: %v", doc.Email, err)
		}

		// Get role_id for 'doctor'
		var roleID uuid.UUID
		err = pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'doctor'").Scan(&roleID)
		if err != nil {
			log.Fatalf("Failed to retrieve role ID for doctor: %v", err)
		}

		// Insert role
		_, err = pool.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			VALUES ($1, $2)
			ON CONFLICT (user_id, role_id) DO NOTHING
		`, userID, roleID)
		if err != nil {
			log.Fatalf("Failed to insert user role for %s: %v", doc.Email, err)
		}

		// Insert doctor profile
		docID := uuid.New()
		specID := specialtyIDs[doc.SpecialtyName]
		_, err = pool.Exec(ctx, `
			INSERT INTO doctors (id, user_id, specialty_id, license_number, is_credential_verified, consultation_fee)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (user_id) DO NOTHING
		`, docID, userID, specID, doc.License, true, doc.Fee)
		if err != nil {
			log.Fatalf("Failed to insert doctor profile for %s: %v", doc.FullName, err)
		}
	}
	fmt.Println("Doctors seeded successfully!")
}
