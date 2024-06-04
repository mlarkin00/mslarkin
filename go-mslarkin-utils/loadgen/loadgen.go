package loadgen

import (
	"log"
	// "strconv"
	"time"
	// goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
)

// type Fibonacci struct {
//     num    float64
//     answer float64
// }

func newFibonacci(n float64, sleepMs int, stopChan chan bool) float64 {

	var answer float64
    c1 := make(chan float64)
    c2 := make(chan float64)

	select {
	case <- stopChan:
		return 0
	default:
		if n <= 1 {
			answer = n
		} else {
			go func() {
				answer := newFibonacci(n - 1, sleepMs, stopChan)
				time.Sleep(time.Duration(sleepMs) * time.Millisecond)
				c2 <- answer
			}()
			go func() {
				answer := newFibonacci(n - 2, sleepMs, stopChan)
				time.Sleep(time.Duration(sleepMs) * time.Millisecond)
				c1 <- answer   
			}()

			answer = <-c2 + <-c1
		}
		close(c1)
		close(c2)
	}

    return answer
}

//1 vCPU 2GB
// targetNum: 30, numCycles: 4 | 43s, ~60%CPU, 45%Memory
// targetNum: 25, numCycles: 6 | 45s, ~70%CPU, 20%Memory
// targetNum: 31, numCycles: 3 | 52s, ~83%CPU, 60%Memory
func LoadGen(targetNum float64, numCycles int, sleepMs int, stopChan chan bool) {
	
	start := time.Now()
	log.Println("Starting: Fibonacci #:", targetNum, "| Cycles:", numCycles, "| sleepMs:", sleepMs)
	for i:=0; i < numCycles; i++ {
		_ = newFibonacci(targetNum, sleepMs, stopChan)
	}
	end := time.Now()
	totalTime := end.Sub(start)
	log.Printf("Complete: Fibonacci #: %v | Cycles: %v | sleepMs: %v | Time: %v", targetNum, numCycles, sleepMs, totalTime)
}