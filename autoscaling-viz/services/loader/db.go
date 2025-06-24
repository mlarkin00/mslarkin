package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
)

func mustConnectDB() *sql.DB {
	var (
		dbUser      = mustGetEnv("DB_USER")
		dbPass      = mustGetEnv("DB_PASS")
		dbName      = mustGetEnv("DB_NAME")
		sqlInstance = mustGetEnv("SQL_INSTANCE")
	)

	dsn := fmt.Sprintf("user=%s password=%s database=%s", dbUser, dbPass, dbName)
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("error parsing dsn config: %s", err)
	}
	var opts []cloudsqlconn.Option

	d, err := cloudsqlconn.NewDialer(context.Background(), opts...)
	if err != nil {
		log.Fatalf("cloudsqlconn.NewDialer: %s", err)
	}

	config.DialFunc = func(ctx context.Context, network, instance string) (net.Conn, error) {
		return d.Dial(ctx, sqlInstance)
	}
	dbURI := stdlib.RegisterConnConfig(config)
	dbPool, err := sql.Open("pgx", dbURI)
	if err != nil {
		log.Fatalf("sql.Open: %s", err)
	}
	dbPool.SetMaxOpenConns(1)
	dbPool.SetConnMaxLifetime(10 * time.Minute)

	pingErr := dbPool.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}

	return dbPool
}
