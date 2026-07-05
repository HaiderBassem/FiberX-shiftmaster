package repository

import (
	"context"
	"time"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type SecurityRepository interface {
	BlockIP(ctx context.Context, ip string, reason string, expiresAt time.Time) error
	IsIPBlocked(ctx context.Context, ip string) (bool, error)
	GetBlockedIPs(ctx context.Context) ([]models.IPBlock, error)
	UnblockIP(ctx context.Context, ip string) error
	CleanupExpiredBlocks(ctx context.Context) error
}

type securityRepo struct {
	db *database.DB
}

func NewSecurityRepository(db *database.DB) SecurityRepository {
	return &securityRepo{db: db}
}

func (r *securityRepo) BlockIP(ctx context.Context, ip string, reason string, expiresAt time.Time) error {
	query := `
		INSERT INTO ip_blocks (ip_address, reason, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (ip_address) DO UPDATE 
		SET reason = EXCLUDED.reason, expires_at = EXCLUDED.expires_at, blocked_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query, ip, reason, expiresAt)
	return err
}

func (r *securityRepo) IsIPBlocked(ctx context.Context, ip string) (bool, error) {
	var count int
	query := `SELECT count(*) FROM ip_blocks WHERE ip_address = $1 AND expires_at > CURRENT_TIMESTAMP`
	err := r.db.QueryRow(ctx, query, ip).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *securityRepo) GetBlockedIPs(ctx context.Context) ([]models.IPBlock, error) {
	query := `SELECT ip_address, reason, blocked_at, expires_at FROM ip_blocks WHERE expires_at > CURRENT_TIMESTAMP ORDER BY blocked_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []models.IPBlock
	for rows.Next() {
		var b models.IPBlock
		if err := rows.Scan(&b.IPAddress, &b.Reason, &b.BlockedAt, &b.ExpiresAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}

func (r *securityRepo) UnblockIP(ctx context.Context, ip string) error {
	query := `DELETE FROM ip_blocks WHERE ip_address = $1`
	_, err := r.db.Exec(ctx, query, ip)
	return err
}

func (r *securityRepo) CleanupExpiredBlocks(ctx context.Context) error {
	query := `DELETE FROM ip_blocks WHERE expires_at <= CURRENT_TIMESTAMP`
	_, err := r.db.Exec(ctx, query)
	return err
}
