package loadgen

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	"google.golang.org/api/iterator"
)

func getCpus() int {

	cgroupMax, _ := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	cgroupPeriod, _ := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	cpuLimitRaw := strings.TrimSpace(string(cgroupMax))
	periodRaw := strings.TrimSpace(string(cgroupPeriod))
	// fmt.Printf("Raw CPU Limit: %s, Period: %s\n", cpuLimitRaw, periodRaw)
	cpuLimit, _ := strconv.ParseFloat(cpuLimitRaw, 32)
	period, _ := strconv.ParseFloat(periodRaw, 32)
	// fmt.Printf("CPU Limit: %v, Period: %v\n", cpuLimit, period)
	vcpus := math.Ceil(cpuLimit / period)
	// fmt.Printf("vCPUs: %v\n", vcpus)
	return int(vcpus)
}

func CpuLoadGen(ctx context.Context, targetPct float64, showLogs bool) {

	availableCpus := getCpus()
	if showLogs {
		log.Printf("Loading %v CPUs at %v%%\n", availableCpus, targetPct)
	}

	// Break down the loadgen into 100ms segments, and load for a % of each segment
	timeUnitMs := float64(100)
	runtimeMs := timeUnitMs * (targetPct / 100)
	sleepMs := timeUnitMs - runtimeMs
	for i := 0; i < availableCpus; i++ {
		go func() {
			runtime.LockOSThread()
			for {
				begin := time.Now()
			PartitionLoop:
				for {
					select {
					case <-ctx.Done():
						return
					default:
						if time.Since(begin) > time.Duration(runtimeMs)*time.Millisecond {
							break PartitionLoop
						}
					}
				}
				time.Sleep(time.Duration(sleepMs) * time.Millisecond)
			}

		}()
	}
	<-ctx.Done()
	if showLogs {
		log.Println("Ending Loadgen")
	}
}

// ////////////////////////////////////////////////////
// Trigger time-bound load with request
// Request params
// targetCpuPct - the % load to generate
// durationS - the duration of the load
// Env Var
// NUM_CPU - the number of available/configured CPUs
// /////////////////////////////////////////////////////
func CpuLoadHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	r = r.WithContext(ctx)

	targetCpuPct, _ := strconv.ParseFloat(goutils.GetParam(r, "targetCpuPct", "5"), 64)
	durationS, _ := strconv.Atoi(goutils.GetParam(r, "durationS", "1"))
	// configCpus, _ := strconv.Atoi(goutils.GetEnv("NUM_CPU", "1"))
	configCpus := getCpus()

	// Use background context to enable request to trigger loadgen without waiting to return response
	loadCtx, loadCtxCancel := context.WithTimeout(context.Background(), time.Duration(durationS)*time.Second)
	defer loadCtxCancel()

	log.Println("Starting Request Load - CPUs:", configCpus, " Pct:", targetCpuPct, " Duration (s):", durationS)

	CpuLoadGen(loadCtx, targetCpuPct, true)
	fmt.Fprintf(w, "Request Load complete\n")
}

func AsyncCpuLoadHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	r = r.WithContext(ctx)

	targetCpuPct, _ := strconv.ParseFloat(goutils.GetParam(r, "targetCpuPct", "5"), 64)
	durationS, _ := strconv.Atoi(goutils.GetParam(r, "durationS", "1"))
	// configCpus, _ := strconv.Atoi(goutils.GetEnv("NUM_CPU", "1"))
	configCpus := getCpus()

	// Use background context to enable request to trigger loadgen without waiting to return response
	loadCtx, loadCtxCancel := context.WithTimeout(context.Background(), time.Duration(durationS)*time.Second)
	defer loadCtxCancel()

	log.Println("Starting Request Load - CPUs:", configCpus, " Pct:", targetCpuPct, " Duration (s):", durationS)

	go CpuLoadGen(loadCtx, targetCpuPct, true)
	fmt.Fprintf(w, "Request Load triggered\n")
}

// ConfigParams holds the configuration parameters from the user input.
// These parameters are used to define a load generation test.
type ConfigParams struct {
	ID string `firestore:"-" json:"id"`
	// TargetURL is the URL of the service to be tested.
	TargetURL string `firestore:"targetUrl" json:"targetUrl"`
	// TargetCPU is the target CPU utilization percentage for the load test.
	TargetCPU int `firestore:"targetCpu,omitempty" json:"targetCpu,omitempty"`
	// QPS is the number of queries per second to be sent to the target URL.
	QPS int `firestore:"qps,omitempty" json:"qps,omitempty"`
	// Duration is the duration of the load test in seconds.
	Duration int `firestore:"duration,omitempty" json:"duration,omitempty"`
}

// projectIDEnv is the environment variable that contains the Google Cloud project ID.
const projectIDEnv = "GOOGLE_CLOUD_PROJECT"

// collectionName is the name of the Firestore collection where the load generation
// configurations are stored.
const collectionName = "loadgen-configs"

var (
	// firestoreClient is the client used to interact with Firestore.
	firestoreClient *firestore.Client
)

func RequestLoadgenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	projectID := os.Getenv(projectIDEnv)
	if projectID == "" {
		projectID = "mslarkin-ext" // Default project ID
	}

	var err error
	firestoreClient, err = firestore.NewClientWithDatabase(ctx, projectID, "loadgen-target-config")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	var configs []ConfigParams
	iter := firestoreClient.Collection(collectionName).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating documents: %v", err)
			http.Error(w, "Failed to retrieve configurations", http.StatusInternalServerError)
			return
		}
		var config ConfigParams
		if err := doc.DataTo(&config); err != nil {
			log.Printf("Error converting document data: %v", err)
			continue
		}
		config.ID = doc.Ref.ID
		configs = append(configs, config)
	}

	if len(configs) == 0 {
		http.Error(w, "No loadgen configurations found", http.StatusInternalServerError)
		return
	}

	config := configs[0]

	targetURL := config.TargetURL + "/loadgen"
	qps := config.QPS
	duration := config.Duration
	targetCPU := config.TargetCPU

	log.Printf("Starting request loadgen: URL=%s, QPS=%d, Duration=%ds, TargetCPU=%d%%", targetURL, qps, duration, targetCPU)

	if qps == 0 {
		log.Println("QPS is 0, no load will be generated.")
		fmt.Fprintf(w, "QPS is 0, no load will be generated.")
		return
	}

	ticker := time.NewTicker(time.Second / time.Duration(qps))
	defer ticker.Stop()

	done := time.After(time.Duration(duration) * time.Second)

	for {
		select {
		case <-done:
			log.Println("Load generation finished.")
			fmt.Fprintf(w, "Load generation finished.")
			return
		case <-ticker.C:
			go func() {
				req, err := http.NewRequest("GET", targetURL, nil)
				if err != nil {
					log.Printf("Error creating request: %v", err)
					return
				}
				q := req.URL.Query()
				q.Add("targetCpuPct", strconv.Itoa(targetCPU))
				req.URL.RawQuery = q.Encode()
				
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Printf("Error sending request: %v", err)
					return
				}
				defer resp.Body.Close()
				log.Printf("Request sent, status: %s", resp.Status)
			}()
		}
	}
}
