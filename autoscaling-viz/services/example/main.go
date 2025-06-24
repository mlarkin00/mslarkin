package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/pubsub"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var projectID string
var revisionID string
var serviceID string
var pubSubTopicName string
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

func init() {
	flag.StringVar(&projectID, "project", "", "Google Cloud project")
	flag.Parse()
	if projectID == "" && metadata.OnGCE() {
		var err error
		projectID, err = metadata.ProjectID()
		if err != nil {
			log.Fatal(err)
		}
	}
	revisionID = os.Getenv("K_REVISION")
	if revisionID == "" {
		revisionID = "K_REVISION_UNSET"
	}

	serviceID = os.Getenv("K_SERVICE")
	if serviceID == "" {
		serviceID = "K_SERVICE_UNSET"
	}
	pubSubTopicName = os.Getenv("PUB_SUB_TOPIC")
	if pubSubTopicName == "" {
		pubSubTopicName = "instances"
	}

}

type Instance struct {
	RevisionID string `json:"revisionID"`
	ServiceID  string `json:"ServiceID"`
	StartEvent bool   `json:"startEvent"`
	StopEvent  bool   `json:"stopEvent"`
}

func mustPublishEvent(i *Instance) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatal(fmt.Errorf("pubsub.NewClient: %v", err))
	}
	defer client.Close()

	eventsTopic := client.Topic(pubSubTopicName)

	message, _ := json.Marshal(i)
	result := eventsTopic.Publish(
		ctx,
		&pubsub.Message{Data: message},
	)
	if err != nil {
		log.Fatal(err)
	}
	result.Get(ctx)
}

func main() {
	ctx := context.Background()

	// Determine port for HTTP service.
	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	// Parse response delay in seconds from env
	responseDelay := 0
	if os.Getenv("RESPONSE_DELAY") != "" {
		var err error
		responseDelay, err = strconv.Atoi(os.Getenv("RESPONSE_DELAY"))
		if err != nil {
			responseDelay = 0
		}
	}

	h2s := &http2.Server{}

	srv := &http.Server{
		Addr: ":" + port,
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wait here to make sure we don't hit QPS limits
			if responseDelay > 0 {
				time.Sleep(time.Second * time.Duration(responseDelay))
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "Hello, this is revision %s", revisionID)
		}), h2s),
	}

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Add artifical startup delay
	startupDelay := 0
	if os.Getenv("STARTUP_DELAY") != "" {
		var err error
		startupDelay, err = strconv.Atoi(os.Getenv("STARTUP_DELAY"))
		if err != nil {
			responseDelay = 0
		}
	}
	if startupDelay > 0 {
		log.Printf("Sleeping for %d seconds...\n", startupDelay)
		time.Sleep(time.Second * time.Duration(startupDelay))
	} else {
		log.Println("Set STARTUP_DELAY in seconds to wait during startup", startupDelay)
	}

	// Start HTTP server.
	go func() {
		log.Printf("listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Now that the app is serving requets, publish start event
	go mustPublishEvent(&Instance{
		RevisionID: revisionID,
		ServiceID:  serviceID,
		StartEvent: true,
		StopEvent:  false,
	})

	sig := <-signalChan
	log.Printf("signal caught: %s", sig)

	mustPublishEvent(&Instance{
		RevisionID: revisionID,
		ServiceID:  serviceID,
		StartEvent: false,
		StopEvent:  true,
	})

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed: %+v", err)
	}
	log.Print("server exited")
	os.Exit(0)
}
