package main

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Response struct {
	Time       time.Time
	StatusCode int
	Duration   time.Duration
}

type Service struct {
	srv           *http.Server
	client        *http.Client
	pool          *sql.DB
	workerPool    *ResizablePool
	instance      *LoaderInstanceRecord
	stopPolling   chan bool
	cancelWorkers context.CancelFunc
}

type LoaderInstanceRecord struct {
	ID          int
	Concurrency int
	Method      string
	Body        string
	Href        string
}

type LoadResult struct {
	TotalNon200   int
	TotalRequests int
	DeltaDuration int
	LastUpdate    time.Time
	PrevTotal     int
}

var mu sync.Mutex
var loadResults = LoadResult{
	TotalNon200:   0,
	TotalRequests: 0,
	PrevTotal:     0,
	DeltaDuration: 0,
	LastUpdate:    time.Now(),
}

var signalChan chan (os.Signal) = make(chan os.Signal, 1)

var reseted = atomic.Bool{}

func (s *Service) removeInstanceRecord() {
	_, err := s.pool.Exec("DELETE FROM loader_instances WHERE id=$1", s.instance.ID)
	if err != nil {
		fmt.Printf("error deleting loader_instances record: %v\n", err)
	}
}

func storeResponse(resp *Response) {
	mu.Lock()
	defer mu.Unlock()

	loadResults.TotalRequests++
	if resp.StatusCode != 200 {
		fmt.Println(resp)
		loadResults.TotalNon200++
	} else {
		loadResults.DeltaDuration += int(resp.Duration.Milliseconds())
	}
}

func (s *Service) checkReset() {
	for {
		time.Sleep(2 * time.Second)
		if reseted.Load() {
			fmt.Println("Reset forced, signing out")
			s.Shutdown()
		}
	}
}

func main() {
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	s := Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	go s.checkReset()

	s.pool = mustConnectDB()

	id, err := s.initInstanceID()
	if err != nil {
		log.Fatalf("no ID: %s", err)
	}
	fmt.Printf("Loader %d\n", id)
	s.instance = &LoaderInstanceRecord{
		ID: id,
	}

	go s.listenHTTP()

	ctx, cancelWorkers := context.WithCancel(context.Background())

	s.cancelWorkers = cancelWorkers
	s.workerPool = NewResizablePool(
		ctx,
		s.request,
	)

	// Start config poller
	go s.startConfigPoller()

	// Shutdown
	sig := <-signalChan
	log.Printf("%s signal caught", sig)

	s.Shutdown()

}

func (s *Service) Shutdown() {
	fmt.Println("Stop config poller")
	s.stopPolling <- true

	fmt.Println("Size pool to 0")
	s.workerPool.Resize(0)

	fmt.Println("Cancel remaining workers")
	s.cancelWorkers()

	fmt.Println("Remove instance record")
	s.removeInstanceRecord()

	if err := s.pool.Close(); err != nil {
		log.Printf("error closing db pool: %+v", err)
	}

	if err := s.srv.Shutdown(context.Background()); err != nil {
		log.Printf("server shutdown failed: %+v", err)
	}

	log.Print("server exited")
	os.Exit(0)
}

func (s *Service) request(ctx context.Context) {
	// To make sure not all the requests arrive at the same time.
	time.Sleep(time.Duration(rand.Intn(250)) * time.Millisecond)
	start := time.Now()

	instance := s.instance

	// Assuming body is json..
	req, err := buildReq(instance)
	if err != nil {
		log.Printf("build request: %s\n", err)
		return
	}

	resp, err := s.client.Do(req)
	if err != nil {
		// Client errors, log and ignore.
		log.Printf("make request: %s\n", err)
	} else {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		duration := time.Since(start)
		storeResponse(&Response{
			Time:       time.Now(),
			StatusCode: resp.StatusCode,
			Duration:   duration,
		})
	}

}

func buildReq(instance *LoaderInstanceRecord) (*http.Request, error) {
	url := instance.Href
	if url == "" {
		return nil, fmt.Errorf("No HREF specified")
	}

	method := instance.Method
	if method == "" {
		method = "GET"
	}

	if instance.Body != "" {
		body := bytes.NewBuffer([]byte(instance.Body))
		req, err := http.NewRequestWithContext(
			context.Background(),
			method,
			url,
			body)
		// Default POST body to JSON
		req.Header.Add("Content-Type", "application/json")
		if err != nil {
			return nil, fmt.Errorf("error while requesting %s: %s\n", url, err)
		}
		return req, nil
	} else { // No body specified
		req, err := http.NewRequestWithContext(
			context.Background(),
			method,
			url,
			nil)
		if err != nil {
			return nil, fmt.Errorf("error while requesting %s: %s\n", url, err)
		}
		return req, nil
	}
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("environment variable %s is not set", key)
	}
	return value
}

func (s *Service) initInstanceID() (int, error) {
	rows, err := s.pool.Query(`INSERT INTO 
		loader_instances (id) 
		VALUES (DEFAULT) 
		RETURNING id`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		var loaderId int
		err := rows.Scan(&loaderId)
		if err != nil {
			return 0, err
		} else {
			return loaderId, nil
		}
	}
	return 0, fmt.Errorf("no ID returned")
}

func (s *Service) persistLoadResults() error {
	mu.Lock()
	defer mu.Unlock()

	deltaDuration := loadResults.DeltaDuration
	loadResults.DeltaDuration = 0
	deltaRequests := loadResults.TotalRequests - loadResults.PrevTotal
	duration := 0.0
	if deltaRequests > 0 {
		duration = float64(deltaDuration / deltaRequests)
	}

	now := time.Now()
	elapsedTime := now.Sub(loadResults.LastUpdate)
	loadResults.PrevTotal = loadResults.TotalRequests
	loadResults.LastUpdate = now
	rate := int(float64(1_000*deltaRequests) / float64(elapsedTime.Milliseconds()))

	_, err := s.pool.Exec(`INSERT INTO loader_request_totals 
			(loader_instance_id, total_requests, failed_requests, rate_per_second, duration)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT  (loader_instance_id)
			DO UPDATE 
				SET total_requests = $2,
			    failed_requests = $3,
				rate_per_second = $4, 
				duration = $5`,
		s.instance.ID,
		loadResults.TotalRequests,
		loadResults.TotalNon200,
		rate,
		duration,
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %s", err)
	}
	return nil
}

/*
This poller will update the href and concurrency record,
and resize the pool if necessary.

It'll also write request count updates.
*/
func (s *Service) startConfigPoller() {
	ticker := time.NewTicker(time.Duration(rand.Intn(250)+500) * time.Millisecond)
	defer ticker.Stop()
	s.stopPolling = make(chan bool)

	for {
		select {
		case <-ticker.C:
			err := s.retrieveAndResize()
			if err != nil {
				fmt.Println(err)
			}
			err = s.persistLoadResults()
			if err != nil {
				fmt.Println(err)
			}
		case <-s.stopPolling:
			return
		}
	}
}

func (s *Service) retrieveAndResize() error {
	var concurrency int
	var reset bool
	rows, err := s.pool.Query(`SELECT concurrency, href, method, body, reset
				FROM loader_instances li 
				CROSS JOIN loader_config 
				WHERE li.id = $1`, s.instance.ID)
	if err != nil {
		s.workerPool.Resize(0)
		return err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(
			&concurrency,
			&s.instance.Href,
			&s.instance.Method,
			&s.instance.Body,
			&reset); err != nil {
			return err
		}
		reseted.Store(reset)
		if concurrency != s.instance.Concurrency {
			s.instance.Concurrency = concurrency
			fmt.Printf(
				"resize %d -> %d\n",
				s.instance.Concurrency,
				concurrency,
			)
			s.workerPool.Resize(concurrency)
		}
	}
	return nil
}

// 	config, err := s.updateInstanceConfig()
// 	if err != nil {
// 		log.Fatalf("error getting current loader config: %+v", err)
// 	}
// 	if config != nil {
// 		if s.loadConfig == nil || s.loadConfig.StartTime != config.StartTime {
// 			fmt.Printf("New active config: %+v\n", config)
// 		}
// 		// Sync starts across intances using the start time.
// 		time.Sleep(time.Until(config.StartTime))

// 		// Asyncronously start to update request totals
// 		// for the duration of the test plus the request timeout

// 		// Execute the load work
// 		s.loadConfig = config
// 		s.workerPool.Resize(config.Concurrency)

// 		// Wait until end of test
// 		time.Sleep(time.Until(config.EndTime))

// 		s.loadConfig = nil
// 		s.workerPool.Resize(0)
// 	} else {
// 		if s.loadConfig != nil {
// 			fmt.Println("No active config")
// 		}
// 		s.loadConfig = nil
// 		s.workerPool.Resize(0)
// 	}
// }
// }

func (s *Service) listenHTTP() {
	// Determine port for HTTP service.
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	h2s := &http2.Server{}

	s.srv = &http.Server{
		Addr: ":" + port,
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// JSON encode config with encoder
			encoder := json.NewEncoder(w)
			encoder.SetIndent("", "  ")
			w.Header().Set("Content-Type", "application/json")
			err := encoder.Encode("OK")
			if err != nil {
				http.Error(w, "Error", http.StatusInternalServerError)
				log.Printf("error while marshaling config: %+v", err)
				return
			}
		}), h2s),
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
