package main

import (
	"context"
	"fmt"
	"os"

	// "regexp"
	// "math"
	"os/signal"
	// "runtime"
	"syscall"
	"time"

	// "io"
	"sync/atomic"
	// gcputils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/gcputils"
	// goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	// loadgen "github.com/mlarkin00/mslarkin/go-mslarkin-utils/loadgen"
	pubsub "cloud.google.com/go/pubsub"
	// run "cloud.google.com/go/run/apiv2"
	// runpb "cloud.google.com/go/run/apiv2/runpb"
	// timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	// monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	// monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	// "github.com/golang/protobuf/ptypes/duration"
	// "github.com/golang/protobuf/ptypes/timestamp"
)

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

func main() {
	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	projectId := "mslarkin-ext"
	subId := "pull-queue-testing"

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			fmt.Println("Waiting for messages")
			msgCount := pullMsgs(ctx, projectId, subId)
			fmt.Printf("Handled %d messages\n", msgCount)
			cancel()
		}
	}()

	// monitoringMetric := "run.googleapis.com/container/instance_count"
	// service := "SERVICE_NAME"
	// region := "REGION"

	// metricFilter := fmt.Sprintf("metric.type=\"%s\"" +
	// 							 " AND resource.labels.service_name =\"%s\"" +
	// 							 " AND resource.labels.location =\"%s\"",
	// 					monitoringMetric, service, region)

	// fmt.Println(metricFilter)

	// runService, err = gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1")
	// udt := gcputils.GetLastUpdateTs("pubsub-pull-subscriber", "mslarkin-ext", "us-central1")
	// fmt.Println(udt)
	// fmt.Println(gcputils.GetInstanceCount("go-worker", "mslarkin-ext", "us-central1"))
	// fmt.Println(gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1"))

	// Receive output from signalChan.
	sig := <-signalChan
	fmt.Printf("%s signal caught\n", sig)

}

func handlePubsubMessage(ctx context.Context, m *pubsub.Message) {
	// fmt.Printf("Got message: %q\n", m.Data)
	m.Ack()
}

func pullMsgs(ctx context.Context, projectId, subId string) int32 {

	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		fmt.Printf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(subId)
	// Must set ReceiveSettings.Synchronous to false (or leave as default) to enable
	// concurrency pulling of messages. Otherwise, NumGoroutines will be set to 1.
	sub.ReceiveSettings.Synchronous = false
	// NumGoroutines determines the number of goroutines sub.Receive will spawn to pull
	// messages.
	sub.ReceiveSettings.NumGoroutines = 4
	// MaxOutstandingMessages limits the number of concurrent handlers of messages.
	// In this case, up to 1 unacked messages can be handled concurrently.
	// Note, even in synchronous mode, messages pulled in a batch can still be handled
	// concurrently.
	sub.ReceiveSettings.MaxOutstandingMessages = 1

	var receivedCount int32 = 0

	// Receive blocks until the context is cancelled or an error occurs.
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		handlePubsubMessage(ctx, msg)
		atomic.AddInt32(&receivedCount, 1)
	})
	if err != nil {
		fmt.Printf("sub.Receive returned error: %v", err)
	}

	return receivedCount
}
