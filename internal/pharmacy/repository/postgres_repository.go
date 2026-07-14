package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/timurdianradhasejati/telemed_hub/internal/pharmacy/model"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, tx pgx.Tx, ord *model.Order) error {
	if ord.ID == uuid.Nil {
		ord.ID = uuid.New()
	}
	now := time.Now().UTC()
	ord.CreatedAt = now
	ord.UpdatedAt = now

	insertOrderQuery := `
		INSERT INTO orders (id, patient_id, prescription_id, status, total_amount, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, insertOrderQuery,
			ord.ID, ord.PatientID, ord.PrescriptionID, ord.Status, ord.TotalAmount,
			ord.CreatedAt, ord.UpdatedAt, ord.CreatedBy,
		)
	} else {
		_, err = r.db.Exec(ctx, insertOrderQuery,
			ord.ID, ord.PatientID, ord.PrescriptionID, ord.Status, ord.TotalAmount,
			ord.CreatedAt, ord.UpdatedAt, ord.CreatedBy,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	insertItemQuery := `
		INSERT INTO order_items (id, order_id, medicine_id, quantity, unit_price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for i := range ord.Items {
		item := &ord.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		item.OrderID = ord.ID
		item.CreatedAt = now
		item.UpdatedAt = now

		if tx != nil {
			_, err = tx.Exec(ctx, insertItemQuery,
				item.ID, item.OrderID, item.MedicineID, item.Quantity, item.UnitPrice,
				item.CreatedAt, item.UpdatedAt,
			)
		} else {
			_, err = r.db.Exec(ctx, insertItemQuery,
				item.ID, item.OrderID, item.MedicineID, item.Quantity, item.UnitPrice,
				item.CreatedAt, item.UpdatedAt,
			)
		}
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	selectOrderQuery := `
		SELECT id, patient_id, prescription_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE id = $1 AND deleted_at IS NULL
	`
	var ord model.Order
	err := r.db.QueryRow(ctx, selectOrderQuery, id).Scan(
		&ord.ID, &ord.PatientID, &ord.PrescriptionID, &ord.Status, &ord.TotalAmount,
		&ord.CreatedAt, &ord.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	selectItemsQuery := `
		SELECT id, order_id, medicine_id, quantity, unit_price, created_at, updated_at
		FROM order_items
		WHERE order_id = $1
	`
	rows, err := r.db.Query(ctx, selectItemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		err = rows.Scan(
			&item.ID, &item.OrderID, &item.MedicineID, &item.Quantity, &item.UnitPrice,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		ord.Items = append(ord.Items, item)
	}

	return &ord, nil
}

func (r *PostgresRepository) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error) {
	selectOrderQuery := `
		SELECT id, patient_id, prescription_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`
	var ord model.Order
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, selectOrderQuery, id).Scan(
			&ord.ID, &ord.PatientID, &ord.PrescriptionID, &ord.Status, &ord.TotalAmount,
			&ord.CreatedAt, &ord.UpdatedAt,
		)
	} else {
		err = r.db.QueryRow(ctx, selectOrderQuery, id).Scan(
			&ord.ID, &ord.PatientID, &ord.PrescriptionID, &ord.Status, &ord.TotalAmount,
			&ord.CreatedAt, &ord.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to fetch order for update: %w", err)
	}

	// Fetch items
	selectItemsQuery := `
		SELECT id, order_id, medicine_id, quantity, unit_price, created_at, updated_at
		FROM order_items
		WHERE order_id = $1
	`
	var rows pgx.Rows
	if tx != nil {
		rows, err = tx.Query(ctx, selectItemsQuery, id)
	} else {
		rows, err = r.db.Query(ctx, selectItemsQuery, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		err = rows.Scan(
			&item.ID, &item.OrderID, &item.MedicineID, &item.Quantity, &item.UnitPrice,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		ord.Items = append(ord.Items, item)
	}

	return &ord, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	now := time.Now().UTC()
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`
	var tag tagWrapper
	var err error

	if tx != nil {
		tag, err = tx.Exec(ctx, query, status, now, id)
	} else {
		tag, err = r.db.Exec(ctx, query, status, now, id)
	}

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

type tagWrapper interface {
	RowsAffected() int64
}

func (r *PostgresRepository) List(ctx context.Context, patientID *uuid.UUID, statusFilter *string, page, limit int) ([]*model.Order, int, error) {
	var whereClauses []string
	var args []any
	argCount := 1

	whereClauses = append(whereClauses, "deleted_at IS NULL")

	if patientID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("patient_id = $%d", argCount))
		args = append(args, *patientID)
		argCount++
	}

	if statusFilter != nil && *statusFilter != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *statusFilter)
		argCount++
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereSQL)
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	if total == 0 {
		return []*model.Order{}, 0, nil
	}

	offset := (page - 1) * limit
	listQuery := fmt.Sprintf(`
		SELECT id, patient_id, prescription_id, status, total_amount, created_at, updated_at
		FROM orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argCount, argCount+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var ord model.Order
		err = rows.Scan(
			&ord.ID, &ord.PatientID, &ord.PrescriptionID, &ord.Status, &ord.TotalAmount,
			&ord.CreatedAt, &ord.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan order: %w", err)
		}

		// Fetch items for this order
		selectItemsQuery := `
			SELECT id, order_id, medicine_id, quantity, unit_price, created_at, updated_at
			FROM order_items
			WHERE order_id = $1
		`
		itemRows, err := r.db.Query(ctx, selectItemsQuery, ord.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch order items in list: %w", err)
		}

		for itemRows.Next() {
			var item model.OrderItem
			err = itemRows.Scan(
				&item.ID, &item.OrderID, &item.MedicineID, &item.Quantity, &item.UnitPrice,
				&item.CreatedAt, &item.UpdatedAt,
			)
			if err != nil {
				itemRows.Close()
				return nil, 0, fmt.Errorf("failed to scan order item in list: %w", err)
			}
			ord.Items = append(ord.Items, item)
		}
		itemRows.Close()

		orders = append(orders, &ord)
	}

	return orders, total, nil
}
