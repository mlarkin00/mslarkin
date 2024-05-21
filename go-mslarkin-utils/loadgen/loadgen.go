package main

import (
	// "fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strconv"
	"runtime"
	"math"
	// "math/rand"
	// "net/http"
	// "context"
)

func Getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

func isPrime(n uint64) bool {
	if n <= 1 {
		return false
	}
	for i := uint64(2); i < n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func computePrimes(cpuBreakMs int, startNum uint64, maxPrimes int) {
	var number uint64
	primeCount := 0
	if startNum%2 == 1 {
		number = startNum
	} else {
		number = startNum - 1
	}

	for primeCount < maxPrimes {
		if isPrime(number) {
			primeCount++
		}
		number += 2
		time.Sleep(time.Duration(cpuBreakMs) * time.Millisecond)
	}
}

type LoadGen struct {
	startDelay  int
	workBreak   int
	cpuBreakMs  int
	startNum    uint64
	maxPrimes   int
	memoryMb    int
	randomBreak bool
	randomCpu   bool
	randomMem   bool
	maxPctLoad float64
}

func DefaultLoadGen() *LoadGen {
	lg := new(LoadGen)
	lg.startDelay = 0
	lg.workBreak = 600
	lg.cpuBreakMs = 800
	lg.startNum = 1e5
	lg.maxPrimes = 5
	lg.memoryMb = 2
	lg.randomBreak = false
	lg.randomCpu = false
	lg.randomMem = false
	lg.maxPctLoad = 0.4
	return lg
}

func BasicLoadGen(
	startDelay int,
	workBreak int,
	cpuBreakMs int,
	maxPrimes int,
	memoryMb int,
	maxPctLoad float64) *LoadGen {
	lg := new(LoadGen)
	lg.startDelay = startDelay
	lg.workBreak = workBreak
	lg.cpuBreakMs = cpuBreakMs
	lg.startNum = 1e5
	lg.maxPrimes = maxPrimes
	lg.memoryMb = memoryMb
	lg.randomBreak = false
	lg.randomCpu = false
	lg.randomMem = false
	lg.maxPctLoad = maxPctLoad
	return lg
}

func (lg LoadGen) StartWorkload() {
	start := time.Now()
	// numCPU := int(math.Floor(float64(runtime.NumCPU()) * lg.maxPctLoad / 2))
	// loadCPU := int(math.Floor(float64(numCPU) * lg.maxPctLoad))
	
	cpChan := make(chan int, loadCPU)
	for cpu := 0; cpu < loadCPU; cpu++ {
		go func() {
			computePrimes(lg.cpuBreakMs, lg.startNum, lg.maxPrimes)
			cpChan <- cpu
		}()
	}
	cpuItr := <- cpChan
	end := time.Now()
	log.Printf("Workload #%v took %v seconds | CPU Break (ms): %v | Max Primes: %v | Memory (MB): %v", cpuItr, end.Sub(start),  lg.cpuBreakMs,
	 lg.maxPrimes, lg.memoryMb)
}

func (lg LoadGen) Run() {
	log.Println("Delaying work loop start for", lg.startDelay, "seconds")
	time.Sleep(time.Duration(lg.startDelay) * time.Second)

	// Add inifinite for loop here
	for {
		// Add in the randomizaion factors here
		log.Printf("Starting work loop: CPU Break (ms): %v | Max Primes: %v | CPU: %v (%v%%, %v vCPUs) | Memory (MB):%v", lg.cpuBreakMs,
			lg.maxPrimes, numCPU, lg.maxPctLoad*100, loadCPU, lg.memoryMb)

		lg.StartWorkload()

		// Sleep for workBreak
		log.Println("Waiting for", lg.workBreak, "seconds before starting next loop.")
		time.Sleep(time.Duration(lg.workBreak) * time.Second)
	}
}

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

// Divide by 2 to account for hyperthreading
var numCPU = int(math.Floor(float64(runtime.NumCPU()) / 2))
var loadCPU int

func main() {

	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	cpuBreak, _ := strconv.Atoi(Getenv("CPU_BREAK_MS", "1000"))
	workBreak, _ := strconv.Atoi(Getenv("WORK_BREAK_S", "600"))
	maxPrimes, _ := strconv.Atoi(Getenv("MAX_PRIMES", "10"))
	maxPctLoad, _ := strconv.ParseFloat(Getenv("MAX_PCT_LOAD", "0.4"), 64)

	log.Printf("Running Worker with %s vCPUs available", strconv.Itoa(numCPU))

	loadGen := BasicLoadGen(0, workBreak, cpuBreak, maxPrimes, 2, maxPctLoad)
	loadCPU = int(math.Floor(float64(numCPU) * loadGen.maxPctLoad))


	// log.Println("Starting calculation with", startNum)
	go func() {
		loadGen.Run()
		// os.Exit(0)
	}()

	// Receive output from signalChan.
	sig := <-signalChan
	if sig.String() == "terminated" {log.Println("SIGTERM received")}
	log.Printf("Signal caught: %s", sig)

	// Add extra handling here to clean up resources, such as flushing logs and
	// closing any database or Redis connections.

	log.Print("Worker exited")
}

// func Mem(addStr string) {
// 	var mb []byte
//     var s [][]byte

//     add, err := strconv.Atoi(addStr)
//     if err != nil {
// 		panic(err)
// 	}
// 	if len(mb) == 0 {
// 		tmp := make([]byte, 1048576)
// 		for i := 0; i < len(tmp); i++ {
// 			tmp[i] = 10
// 		}
// 		mb = tmp
// 	}
// 	for i := 0; i < add; i++ {
// 		dst := make([]byte, len(mb))
// 		copy(dst, mb)
// 		s = append(s, dst)
// 	}
// }

// func main() {
// 	log.Print("starting server...")
// 	http.HandleFunc("/", handler)
// 	http.HandleFunc("/prj", prjHandler)

// 	// Determine port for HTTP service.
// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "8080"
// 		log.Printf("defaulting to port %s", port)
// 	}

// 	// Start HTTP server.
// 	log.Printf("listening on port %s", port)
// 	if err := http.ListenAndServe(":"+port, nil); err != nil {
// 		log.Fatal(err)
// 	}
// }

// func handler(w http.ResponseWriter, r *http.Request) {
// 	name := os.Getenv("NAME")
// 	if name == "" {
// 		name = "World"
// 	}
// 	fmt.Fprintf(w, "Hello %s!\n", name)
// }

// func prjHandler(w http.ResponseWriter, r *http.Request) {
// 	prj := getProjectId()
// 	fmt.Println("Returned ProjectID: ", prj)
// }

// func getProjectId() string {
// 	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
// 	if err != nil {
// 		panic(err)
// 	}
// 	req.Header.Add("Metadata-Flavor", "Google")
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode == http.StatusOK {
// 		bodyBytes, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			panic(err)
// 		}
// 		bodyString := string(bodyBytes)
// 		fmt.Println(bodyString)
// 		return bodyString
// 	} else {
// 		return string(resp.StatusCode)
// 	}

// }
