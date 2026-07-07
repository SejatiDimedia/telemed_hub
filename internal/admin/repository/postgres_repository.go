package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/admin/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListAuditLogs(ctx context.Context, filter dto.ListAuditLogsFilter) ([]*model.AuditLog, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argCount := 1

	if filter.ActorID != nil && *filter.ActorID != "" {
		actorID, err := uuid.Parse(*filter.ActorID)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid actor_id filter: %w", err)
		}
		where += fmt.Sprintf(" AND actor_id = $%d", argCount)
		args = append(args, actorID)
		argCount++
	}

	if filter.Action != nil && *filter.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, *filter.Action)
		argCount++
	}

	if filter.TargetType != nil && *filter.TargetType != "" {
		where += fmt.Sprintf(" AND target_type = $%d", argCount)
		args = append(args, *filter.TargetType)
		argCount++
	}

	if filter.From != nil && *filter.From != "" {
		fromTime, err := time.Parse("2006-01-02", *filter.From)
		if err != nil {
			fromTime, err = time.Parse(time.RFC3339, *filter.From)
		}
		if err == nil {
			where += fmt.Sprintf(" AND created_at >= $%d", argCount)
			args = append(args, fromTime.UTC())
			argCount++
		}
	}

	if filter.To != nil && *filter.To != "" {
		toTime, err := time.Parse("2006-01-02", *filter.To)
		if err != nil {
			toTime, err = time.Parse(time.RFC3339, *filter.To)
		}
		if err == nil {
			// Include the whole day if parsing yyyy-mm-dd
			if len(*filter.To) == 10 {
				toTime = toTime.Add(24 * time.Hour)
			}
			where += fmt.Sprintf(" AND created_at < $%d", argCount)
			args = append(args, toTime.UTC())
			argCount++
		}
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", where)
	var totalItems int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT id, actor_id, action, target_type, target_id, ip_address, user_agent, metadata, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		err := rows.Scan(
			&l.ID, &l.ActorID, &l.Action, &l.TargetType, &l.TargetID,
			&l.IPAddress, &l.UserAgent, &l.Metadata, &l.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, &l)
	}

	return logs, totalItems, nil
}
