package main

import (
	// "fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strconv"
	// "math/rand"
	// "net/http"
	// "context"
	// "runtime"
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
}

func DefaultLoadGen() *LoadGen {
	lg := new(LoadGen)
	lg.startDelay = 0
	lg.workBreak = 600
	lg.cpuBreakMs = 800
	lg.startNum = 1e13
	lg.maxPrimes = 5
	lg.memoryMb = 2
	lg.randomBreak = false
	lg.randomCpu = false
	lg.randomMem = false
	return lg
}

func BasicLoadGen(
	startDelay int,
	workBreak int,
	cpuBreakMs int,
	maxPrimes int,
	memoryMb int) *LoadGen {
	lg := new(LoadGen)
	lg.startDelay = startDelay
	lg.workBreak = workBreak
	lg.cpuBreakMs = cpuBreakMs
	lg.startNum = 1e10
	lg.maxPrimes = maxPrimes
	lg.memoryMb = memoryMb
	lg.randomBreak = false
	lg.randomCpu = false
	lg.randomMem = false
	return lg
}

func (lg LoadGen) StartWorkload() {
	start := time.Now()
	computePrimes(lg.cpuBreakMs, lg.startNum, lg.maxPrimes)
	end := time.Now()
	log.Println("Workload took", end.Sub(start), "seconds. CPU Break (ms):", lg.cpuBreakMs,
		"| Max Primes:", lg.maxPrimes, "| Memory (MB):", lg.memoryMb)
}

func (lg LoadGen) Run() {
	log.Println("Delaying work loop start for", lg.startDelay, "seconds")
	time.Sleep(time.Duration(lg.startDelay) * time.Second)

	// Add inifinite for loop here
	for {
		// Add in the randomizaion factors here
		log.Println("Starting work loop: CPU Break (ms):", lg.cpuBreakMs,
			"| Max Primes:", lg.maxPrimes, "| Memory (MB):", lg.memoryMb)

		lg.StartWorkload()

		// Sleep for workBreak
		log.Println("Waiting for", lg.workBreak, "seconds before starting next loop.")
		time.Sleep(time.Duration(lg.workBreak) * time.Second)
	}
}

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

func main() {

	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// defaultCpuBreakMs := 350
	// startNum := 1e5 //1e13
	// defaultMaxPrimes := 25
	// loadGen := DefaultLoadGen()
	cpuBreak, _ := strconv.Atoi(Getenv("CPU_BREAK_MS", "1000"))
	workBreak, _ := strconv.Atoi(Getenv("WORK_BREAK_S", "600"))
	maxPrimes, _ := strconv.Atoi(Getenv("MAX_PRIMES", "10"))

	loadGen := BasicLoadGen(0, workBreak, cpuBreak, maxPrimes, 2)

	// log.Println("Starting calculation with", startNum)
	go func() {
		loadGen.Run()
		// computePrimes(defaultCpuBreakMs, uint64(startNum), defaultMaxPrimes)
		// log.Println("Finished calculation with", startNum)
		os.Exit(0)
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
