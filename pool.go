package common

import (
	"context"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// NewPool creates a pgxpool.Pool with observability hooks:
// - Lifecycle hooks (BeforeConnect, AfterConnect, BeforeClose) log events and
//   flip an atomic trackStats flag.
// - BeforeAcquire checks the flag and, if set, logs pool stats once then clears
//   it. Result: stats are logged only when pool topology changes, never during
//   steady state.
func NewPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	var trackStats atomic.Bool
	var poolRef *pgxpool.Pool

	cfg.BeforeConnect = func(ctx context.Context, cc *pgx.ConnConfig) error {
		log.Info().Msg("[pgxpool] BeforeConnect")
		trackStats.Store(true)
		return nil
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		log.Info().Uint32("pid", uint32(conn.PgConn().PID())).Msg("[pgxpool] AfterConnect")
		trackStats.Store(true)
		return nil
	}
	cfg.BeforeClose = func(conn *pgx.Conn) {
		log.Info().Uint32("pid", uint32(conn.PgConn().PID())).Msg("[pgxpool] BeforeClose")
		trackStats.Store(true)
	}
	cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		if trackStats.Swap(false) && poolRef != nil {
			printPoolStats(poolRef)
		}
		return true
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	poolRef = pool
	return pool, nil
}

func printPoolStats(pool *pgxpool.Pool) {
	s := pool.Stat()
	log.Info().
		Int32("max", s.MaxConns()).
		Int32("total", s.TotalConns()).
		Int32("constructing", s.ConstructingConns()).
		Int32("in_use", s.AcquiredConns()).
		Int32("idle", s.IdleConns()).
		Int32("not_connected", s.MaxConns()-s.TotalConns()).
		Msg("[pgxpool] stats")
}
