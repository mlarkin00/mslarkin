package main

import (
	"fmt"
	"log"
	"strconv"

	// "math/rand"
	"net/http"
	"os"

	// "strconv"
	"time"
	goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
)

type Fibonacci struct {
    num    float64
    answer float64
}

func newFibonacci(n float64) *Fibonacci {

    f := new(Fibonacci)
    f.num = n
    c1 := make(chan float64)
    c2 := make(chan float64)

    if f.num <= 1 {
        f.answer = n
    } else {
        go func() {
            fib1 := newFibonacci(n - 1)
            c2 <- fib1.answer
        }()
        go func() {
            fib2 := newFibonacci(n - 2)
            c1 <- fib2.answer   
        }()

        f.answer = <-c2 + <-c1
    }
    close(c1)
    close(c2)

    return f
}

func fibHandler(w http.ResponseWriter, r *http.Request) {
	var f *Fibonacci

	targetNum, _ := strconv.ParseFloat(goutils.GetParam(r, "targetNum", "30"), 64)
	numCycles, _ := strconv.Atoi(goutils.GetParam(r, "numCycles", "4"))
	sleepMs, _ := strconv.Atoi(goutils.GetParam(r, "sleepMs", "0"))

	start := time.Now()
	log.Println("Getting the", targetNum, "th fibonacci number", numCycles,"times.")
	for i:=0; i < numCycles; i++ {
		f = newFibonacci(targetNum)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		// fmt.Println("The",targetNum, "Fibonacci number is:",f.answer)
	}
	end := time.Now()
	totalTime := end.Sub(start)
	log.Printf("The %vth Fibonacci number is %v, and took %v to find %v times", targetNum, f.answer, totalTime, numCycles)
	fmt.Fprintf(w, "The %vth Fibonacci number is %v, and took %v to find %v times", targetNum, f.answer, totalTime, numCycles)
}