package main

import (
	// "context"
	"fmt"
	"os"

	// "regexp"
	// "math"
	"os/signal"
	// "runtime"
	"syscall"
	"time"

	// "io"
	// "sync/atomic"
	gcputils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/gcputils"
	// goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	// loadgen "github.com/mlarkin00/mslarkin/go-mslarkin-utils/loadgen"
	// pubsub "cloud.google.com/go/pubsub"
	// run "cloud.google.com/go/run/apiv2"
	// runpb "cloud.google.com/go/run/apiv2/runpb"
	// timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	// monitoring "cloud.google.com/go/monitoring/apiv3/v2"
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
	topicId := "pull-test"
	// subId := "pull-queue-testing"

	go func() {
		for {
			getPubsubPublishRate(topicId, projectId)
			time.Sleep(10 * time.Second)
		}
	}()

	// runService, err = gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1")
	// udt := gcputils.GetLastUpdateTs("pubsub-pull-subscriber", "mslarkin-ext", "us-central1")
	// fmt.Println(udt)
	// fmt.Println(gcputils.GetInstanceCount("go-worker", "mslarkin-ext", "us-central1"))
	// fmt.Println(gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1"))

	// Receive output from signalChan.
	sig := <-signalChan
	fmt.Printf("%s signal caught\n", sig)

}

func getPubsubPublishRate(topicId string, projectId string) {

	monitoringMetric := "pubsub.googleapis.com/topic/send_request_count"
	aggregationSeconds := 60
	intervalSeconds := 240
	groupBy := []string{"resource.labels.topic_id"}
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.topic_id =\"%s\"",
		monitoringMetric, topicId)
	metricData := gcputils.GetMetricRate(monitoringMetric,
		metricFilter,
		intervalSeconds,
		aggregationSeconds,
		groupBy,
		projectId)

	// fmt.Println(metricData)
	for _, metric := range metricData {
		fmt.Printf("Timestamp: %v - Rate: %v\n",
			metric.Interval.StartTime,
			metric.GetValue().GetDoubleValue())
	}

}
