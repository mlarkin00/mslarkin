package main

import (
	"context"
	"fmt"
	"os"
	// "regexp"
	// "math"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	// gcputils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/gcputils"
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
	// fmt.Println(gcputils.GetInstanceCount("go-worker", "mslarkin-ext", "us-central1"))
	// fmt.Println(gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1"))

	loadCtx, loadCtxCancel := context.WithCancel(context.Background())
	defer loadCtxCancel()

	numCpus := runtime.NumCPU()
	fmt.Println(numCpus, "CPUs available")

	targetCpuPct := float64(50) //strconv.Atoi(goutils.GetParam(r, "targetCpuPct", "40"))
	durationS := 10             //strconv.Atoi(goutils.GetParam(r, "durationS", "10"))

	tgtPct := targetCpuPct / 100
	targetCpus = 1 //int(math.Round(float64(numCpus) * tgtPct))
	fmt.Println("Total CPUs:", numCpus, " Target Pct:", targetCpuPct, ", %:", tgtPct, " Target CPUs:", targetCpus, " Duration:", durationS)

	go func() {
		defer loadCtxCancel()
		for {
			fmt.Println("Starting loop...")
			// fmt.Printf("Context: %v | updateLoad: %v | CPUs: %v", loadCtx, updateLoad, targetCpus)
			// go cpuLoadGen(loadCtx, updateLoad, targetCpus)
			varTest(loadCtx, &updateLoad, targetCpus)

			fmt.Println("Loading...")
			time.Sleep(time.Duration(durationS) * time.Second)
			fmt.Println("Updating...")
			targetCpus++ 
			updateLoad = true
			// fmt.Println("Second Waiting...")
			// fmt.Printf("Context: %v | updateLoad: %v | CPUs: %v", loadCtx, updateLoad, targetCpus)
			// time.Sleep(time.Duration(durationS) * time.Second)
		}
	}()

	// for {
	// 	select {
	// 	case <- loadCtx.Done():
	// 		fmt.Println("Exiting due to Context")
	// 		return
	// 	case sig := <-signalChan:
	// 		fmt.Printf("%s signal caught", sig)
	// 		return
	// 	}
	// }

	sig := <-signalChan
	fmt.Printf("%s signal caught", sig)
	// close(done)

}

func varTest(ctx context.Context, updateLoad *bool, targetCpus int) {
	fmt.Println("Testing CPUs:", targetCpus)
	for i := 0; i < targetCpus; i++ {
		go func() {
			for {
				if *updateLoad {
					fmt.Println("Exiting: Update")
					return
				} else {
					select {
					case <-ctx.Done():
						// exitReason = "context.Done()"
						fmt.Println("Exiting: Context")
						return
					default:
					}
				}
			}
		}()
	}
	// fmt.Println("Leaving test function")
}

func cpuLoadGen(ctx context.Context, updateLoad chan bool, targetCpus int) {
	fmt.Println("Loading", targetCpus, "CPUs")
	exitReason := "default"
	for i := 0; i < targetCpus; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					exitReason = "context.Done()"
					fmt.Println("Exiting: Context")
					return
				case <-updateLoad:
					exitReason = "updateLoad"
					fmt.Println("Exiting: Update")
					return
				default:
				}
			}
		}()
	}
	fmt.Println(exitReason)
}
