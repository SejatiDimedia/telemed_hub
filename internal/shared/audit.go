package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEntry represents a record to be written to audit_logs.
type AuditEntry struct {
	ActorID    uuid.UUID
	Action     string         // e.g., "doctor.verified"
	TargetType string         // e.g., "doctors"
	TargetID   uuid.UUID
	IPAddress  string
	UserAgent  string
	Metadata   map[string]any
}

// AuditService defines the single write-path interface for audit logging.
type AuditService interface {
	Log(ctx context.Context, entry AuditEntry) error
}

// AuditServiceImpl implements AuditService using PostgreSQL.
type AuditServiceImpl struct {
	db *pgxpool.Pool
}

// NewAuditService constructs a new AuditServiceImpl.
func NewAuditService(db *pgxpool.Pool) *AuditServiceImpl {
	return &AuditServiceImpl{db: db}
}

// Log writes an audit log entry to the database.
func (s *AuditServiceImpl) Log(ctx context.Context, entry AuditEntry) error {
	query := `
		INSERT INTO audit_logs (id, actor_id, action, target_type, target_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	id := uuid.New()
	createdAt := time.Now().UTC()

	_, err := s.db.Exec(ctx, query,
		id,
		entry.ActorID,
		entry.Action,
		entry.TargetType,
		entry.TargetID,
		entry.IPAddress,
		entry.UserAgent,
		entry.Metadata,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit log entry: %w", err)
	}

	return nil
}
