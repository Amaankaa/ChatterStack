package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v3"
)

// pgxMockPool adapts pgxmock.PgxPoolIface to the local PgxPool interface.
type pgxMockPool struct {
	pgxmock.PgxPoolIface
}

func newPgxMockPool(t testing.TB) (*pgxMockPool, pgxmock.PgxPoolIface) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgx mock pool: %v", err)
	}
	if t != nil {
		t.Cleanup(func() { m.Close() })
	}
	return &pgxMockPool{PgxPoolIface: m}, m
}

func (p *pgxMockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return p.PgxPoolIface.BeginTx(ctx, pgx.TxOptions{})
}

// Ensure pgxMockPool satisfies the repository pool interface.
var _ PgxPool = (*pgxMockPool)(nil)
