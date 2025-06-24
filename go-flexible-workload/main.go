package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	loadgen "github.com/mlarkin00/mslarkin/go-mslarkin-utils/loadgen"
)

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

func main() {

	startDelay, _ := strconv.Atoi(goutils.GetEnv("COLD_START_DELAY_S", "0"))
	fmt.Printf("Delaying startup by %vs\n", startDelay)
	time.Sleep(time.Duration(startDelay) * time.Second)

	var entrypointMux *http.ServeMux

	ingressPort := os.Getenv("PORT")
	if ingressPort == "" {
		ingressPort = "8080"
		log.Printf("defaulting to port %s", ingressPort)
	} else {
		log.Printf("Listening o port %s", ingressPort)
	}

	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up ingress handlers
	entrypointMux = http.NewServeMux()
	entrypointMux.HandleFunc("/", helloHandler)
	entrypointMux.HandleFunc("/hello", helloHandler)
	entrypointMux.HandleFunc("/loadgen", loadgen.CpuLoadHandler)
	entrypointMux.HandleFunc("/loadgen-async", loadgen.AsyncCpuLoadHandler)
	entrypointMux.HandleFunc("/startupcheck", startupCheckHandler)
	entrypointMux.HandleFunc("/healthcheck", healthCheckHandler)

	go http.ListenAndServe(":"+ingressPort, entrypointMux)

	// Start background load, if configured
	if goutils.GetEnv("BG_LOAD", "False") == "True" {
		loadCpuPct, _ := strconv.ParseFloat(goutils.GetEnv("LOAD_CPU_PCT", "25"), 64)
		configCpus, _ := strconv.Atoi(goutils.GetEnv("NUM_CPU", "1"))

		if configCpus > 0 && loadCpuPct > 0 {
			log.Printf("Starting background CPU loadgen (CPUs: %v, Pct: %v%%)", configCpus, loadCpuPct)
			loadCtx, loadCtxCancel := context.WithCancel(context.Background())
			defer loadCtxCancel()
			go loadgen.CpuLoadGen(loadCtx, loadCpuPct, false)
		}
	}

	sig := <-signalChan
	log.Printf("%s signal caught", sig)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello\n")
}

func startupCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Startup complete\n")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Healthcheck complete\n")
}
