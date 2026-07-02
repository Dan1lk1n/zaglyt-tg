package repository

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"zaglyt-tg/configs"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDB(cfg *configs.Config) (*sqlx.DB, error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	} else {
		dsn = ensureSSLMode(dsn)
	}

	db, err := sqlx.Connect(cfg.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("db connection error: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// ensureSSLMode defaults a user-supplied DATABASE_URL to sslmode=disable when it
// does not specify one, matching the DB_* path (which always disables SSL). This
// avoids "pq: SSL is not enabled on the server" against a plain Postgres. An
// explicit sslmode in the DSN is preserved.
func ensureSSLMode(dsn string) string {
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		q := u.Query()
		if q.Get("sslmode") == "" {
			q.Set("sslmode", "disable")
			u.RawQuery = q.Encode()
		}
		return u.String()
	}

	// Keyword/value DSN form (e.g. "host=... user=...").
	if !strings.Contains(dsn, "sslmode=") {
		return dsn + " sslmode=disable"
	}
	return dsn
}
