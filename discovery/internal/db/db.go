package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func DSN(host, port, user, pass, dbname string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		urlenc(user), urlenc(pass), host, port, dbname)
}

func urlenc(s string) string {
	return strings.ReplaceAll(s, "#", "%23")
}

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(10 * time.Minute)

	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(c); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
