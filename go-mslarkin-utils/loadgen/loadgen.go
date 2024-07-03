package loadgen

import (
	"log"
	"fmt"
	"time"
	"context"
	"runtime"
	"net/http"
	"strconv"
	goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
)

func CpuLoadGen(ctx context.Context, availableCpus int, targetPct float64) {
	log.Printf("Loading %v CPUs at %v%%\n", availableCpus, targetPct)

	timeUnitMs := float64(100)
	runtimeMs := timeUnitMs * targetPct
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
	log.Println("Ending Loadgen...")
}

func CpuLoadHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	r = r.WithContext(ctx)

	targetCpuPct, _ := strconv.ParseFloat(goutils.GetParam(r, "targetCpuPct", "40"), 64)
	durationS, _ := strconv.Atoi(goutils.GetParam(r, "durationS", "60"))
	configCpus, _ := strconv.Atoi(goutils.GetEnv("NUM_CPU", "1"))

	loadCtx, _ := context.WithTimeout(ctx, time.Duration(durationS)*time.Second)

	log.Println("Starting Request Load - CPUs:", configCpus, " Pct:", targetCpuPct, " Duration:", durationS)

	go CpuLoadGen(loadCtx, configCpus, targetCpuPct)
	// log.Println("Ending Request Load")
	fmt.Fprintf(w, "Request Load triggered\n")
}