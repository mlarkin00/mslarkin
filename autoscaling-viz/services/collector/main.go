package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var pool *sql.DB

var signalChan chan (os.Signal) = make(chan os.Signal, 1)

type InstanceEvent struct {
	ID         string    `json:"ID"`
	Time       time.Time `json:"time"`
	RevisionID string    `json:"revisionID"`
	ServiceID  string    `json:"serviceID"`
	StartEvent bool      `json:"startEvent"`
	StopEvent  bool      `json:"stopEvent"`
}

func main() {
	ctx := context.Background()

	// Determine port for HTTP service.
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	h2s := &http2.Server{}

	pool = mustConnectDB()
	defer pool.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/instances", instances)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: h2c.NewHandler(mux, h2s),
	}

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server.
	go func() {
		log.Printf("listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := <-signalChan
	log.Printf("%s signal caught", sig)

	if err := srv.Shutdown(ctx); err != nil {
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
	dbPool.SetMaxOpenConns(80)
	dbPool.SetConnMaxLifetime(10 * time.Minute)

	pingErr := dbPool.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}

	return dbPool
}

// persistEvent persists an instance event to the database.
func persistEvent(ctx context.Context, db *sql.DB, e InstanceEvent) error {
	stmt := `INSERT INTO 
		collector_events (id, time, revision_id, service_id, instance_delta) 
		VALUES ($1, $2, $3, $4, $5)`
	instanceDelta := 0
	if e.StartEvent {
		instanceDelta = 1
	} else if e.StopEvent {
		instanceDelta = -1
	}
	_, err := db.ExecContext(ctx, stmt,
		e.ID, e.Time, e.RevisionID, e.ServiceID, instanceDelta)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func instances(w http.ResponseWriter, r *http.Request) {
	// Read instance struct from POST body
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are accepted", http.StatusMethodNotAllowed)
		return
	}
	var instanceEvent InstanceEvent
	if err := json.NewDecoder(r.Body).Decode(&instanceEvent); err != nil {
		http.Error(w, "Error occurred", http.StatusBadRequest)
		log.Printf("error parsing request body: %+v", err)
		return
	}
	fmt.Println(r.Header)
	messageID := r.Header.Get("X-Goog-Pubsub-Message-Id")
	if messageID == "" {
		http.Error(w, "Error occurred", http.StatusBadRequest)
		log.Printf("X-Goog-Pubsub-Message-Id header is missing")
		return
	}
	instanceEvent.ID = messageID

	publishTime := r.Header.Get("X-Goog-Pubsub-Publish-Time")
	if publishTime == "" {
		http.Error(w, "Error occurred", http.StatusBadRequest)
		log.Printf("X-Goog-Pubsub-Publish-Time header is missing")
		return
	}
	var err error
	instanceEvent.Time, err = time.Parse(time.RFC3339, publishTime)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusBadRequest)
		log.Printf("X-Goog-Pubsub-Publish-Time has the wrong format (not RFC3339)")
	}
	persistEvent(context.Background(), pool, instanceEvent)
}
