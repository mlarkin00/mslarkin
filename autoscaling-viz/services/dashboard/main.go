package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
)

//go:embed public/dist/*
var staticFiles embed.FS

type Service struct {
	srv  *http.Server
	pool *sql.DB
}

type AppConfig struct {
	ShowContainers bool `json:"showContainers"`
}

var signalChan chan (os.Signal) = make(chan os.Signal, 1)

var upgrader = websocket.Upgrader{} // use default options

func main() {
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	s := Service{}

	// Connect DB
	s.pool = mustConnectDB()

	// Listen for HTTP requests
	go s.listenHTTP()

	// Shutdown
	sig := <-signalChan
	log.Printf("signal caught: %s", sig)

	if err := s.pool.Close(); err != nil {
		log.Printf("error closing db pool: %+v", err)
	}

	if err := s.srv.Shutdown(context.Background()); err != nil {
		log.Printf("server shutdown failed: %+v", err)
	}
	log.Print("server exited")
	os.Exit(0)
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("environment variable %s is not set", key)
	}
	return value
}

func (s *Service) listenHTTP() {
	// Determine port for HTTP service.
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	showContainers := false
	if os.Getenv("SHOW_CONTAINERS") == "1" {
		showContainers = true
	}

	// Serve static assets
	var staticFS = fs.FS(staticFiles)
	htmlContent, err := fs.Sub(staticFS, "public/dist")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.FS(htmlContent))

	// Create mux
	mux := http.NewServeMux()

	mux.Handle("/", fs)

	mux.HandleFunc("/api/ws", s.LiveUpdates)
	mux.HandleFunc("/api/start/", s.StartTest)
	mux.HandleFunc("/api/reset", s.Reset)
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&AppConfig{ShowContainers: showContainers})
	})

	s.srv = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}

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
	dbPool.SetMaxOpenConns(5)
	dbPool.SetConnMaxLifetime(10 * time.Minute)

	pingErr := dbPool.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}

	return dbPool
}
