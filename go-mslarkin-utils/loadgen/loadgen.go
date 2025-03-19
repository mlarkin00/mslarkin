package loadgen

import (
	"context"
	"fmt"
	goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
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
	loadCtx, _ := context.WithTimeout(context.Background(), time.Duration(durationS)*time.Second)

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
	loadCtx, _ := context.WithTimeout(context.Background(), time.Duration(durationS)*time.Second)

	log.Println("Starting Request Load - CPUs:", configCpus, " Pct:", targetCpuPct, " Duration (s):", durationS)

	go CpuLoadGen(loadCtx, targetCpuPct, true)
	fmt.Fprintf(w, "Request Load triggered\n")
}
