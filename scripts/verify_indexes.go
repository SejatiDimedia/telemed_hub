package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://telemedhub:telemedhub_secret@localhost:5432/telemedhub?sslmode=disable"
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Printf("ERROR: failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	fmt.Println("Querying active database indexes...")

	rows, err := conn.Query(ctx, `
		SELECT indexname, indexdef 
		FROM pg_indexes 
		WHERE schemaname = 'public'
	`)
	if err != nil {
		fmt.Printf("ERROR: failed to query pg_indexes: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	activeIndexes := make(map[string]string)
	for rows.Next() {
		var indexname, indexdef string
		if err := rows.Scan(&indexname, &indexdef); err != nil {
			fmt.Printf("ERROR: failed to scan row: %v\n", err)
			os.Exit(1)
		}
		activeIndexes[indexname] = indexdef
	}

	// List of recommended indexes to verify
	expectedIndexes := []string{
		"users_email_key", // Postgres auto-generated unique index for email UNIQUE constraint
		"uq_users_phone_number_active",
		"idx_doctors_specialty",
		"idx_doctor_availability_query",
		"uq_appointments_availability_active",
		"idx_appointments_patient_status",
		"idx_appointments_doctor_schedule",
		"idx_consultations_status",
		"idx_wallet_transactions_query",
		"idx_wallet_transactions_idempotency",
		"idx_medical_records_patient_record_type",
		"idx_notifications_user_status",
		"idx_notifications_status_retry",
		"idx_audit_logs_target",
		"idx_audit_logs_actor",
		"idx_refresh_tokens_user_id",
		"idx_refresh_tokens_user_active",
		"idx_ai_sessions_patient_status",
		"idx_ai_suggestions_session_id",
	}

	fmt.Println("\nIndex verification report against docs/06-database-design.md:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-42s | %-12s | %s\n", "Index Name", "Status", "Notes / Definition")
	fmt.Println(strings.Repeat("-", 80))

	allPassed := true
	for _, name := range expectedIndexes {
		def, ok := activeIndexes[name]
		if ok {
			// clean up definition for printing
			defClean := strings.TrimPrefix(def, "CREATE INDEX ")
			defClean = strings.TrimPrefix(defClean, "CREATE UNIQUE INDEX ")
			fmt.Printf("%-42s | \033[32m%-12s\033[0m | %s\n", name, "PASSED", defClean)
		} else {
			// Some versions/setups might name the constraint unique index differently, let's do a fallback check
			fallbackPassed := false
			if name == "users_email_key" {
				for activeName, activeDef := range activeIndexes {
					if strings.Contains(activeName, "users") && strings.Contains(activeDef, "(email)") {
						fmt.Printf("%-42s | \033[32m%-12s\033[0m | %s (Matched via %s)\n", name, "PASSED", activeDef, activeName)
						fallbackPassed = true
						break
					}
				}
			}
			if !fallbackPassed {
				fmt.Printf("%-42s | \033[31m%-12s\033[0m | MISSING!\n", name, "FAILED")
				allPassed = false
			}
		}
	}
	fmt.Println(strings.Repeat("-", 80))

	if allPassed {
		fmt.Println("\033[32mSUCCESS: All recommended indexes are present and verified!\033[0m")
		os.Exit(0)
	} else {
		fmt.Println("\033[31mWARNING: Some indexes are missing or failed verification!\033[0m")
		os.Exit(1)
	}
}
