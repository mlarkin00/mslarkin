package loadgen

import (
	"log"
	// "strconv"
	"time"
	// goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
)

type Fibonacci struct {
    num    float64
    answer float64
}

func newFibonacci(n float64, sleepMs int) *Fibonacci {

    f := new(Fibonacci)
    f.num = n
    c1 := make(chan float64)
    c2 := make(chan float64)

    if f.num <= 1 {
        f.answer = n
    } else {
        go func() {
            fib1 := newFibonacci(n - 1, sleepMs)
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
            c2 <- fib1.answer
        }()
        go func() {
            fib2 := newFibonacci(n - 2, sleepMs)
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
            c1 <- fib2.answer   
        }()

        f.answer = <-c2 + <-c1
    }
    close(c1)
    close(c2)

    return f
}

func LoadGen(targetNum float64, numCycles int, sleepMs int) {

	start := time.Now()
	log.Println("Starting: Fibonacci #:", targetNum, "| Cycles:", numCycles, "| sleepMs:", sleepMs)
	for i:=0; i < numCycles; i++ {
		_ = newFibonacci(targetNum, sleepMs)
	}
	end := time.Now()
	totalTime := end.Sub(start)
	log.Printf("Complete: Fibonacci #: %v | Cycles: %v | Time: %v | sleepMs: %v", targetNum, numCycles, totalTime, sleepMs)
}