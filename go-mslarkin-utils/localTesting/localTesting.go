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
	// "time"
	gcputils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/gcputils"
	// goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	// loadgen "github.com/mlarkin00/mslarkin/go-mslarkin-utils/loadgen"
	// pubsub "google.golang.org/api/pubsub/v1"
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

// var updateLoad chan (bool) = make(chan bool)
var updateLoad bool = false
var targetCpus int

func main() {
	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// runService, err = gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1")
	// udt := gcputils.GetLastUpdateTs("pubsub-pull-subscriber", "mslarkin-ext", "us-central1")
	// fmt.Println(udt)
	fmt.Println(gcputils.GetInstanceCount("go-worker", "mslarkin-ext", "us-central1"))
	// fmt.Println(gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1"))

	
}
