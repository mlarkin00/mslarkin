package goutils

import (
	// "context"
	// "fmt"
	// "io"
	"math"
	"net/http"
	"net/url"
	"os"
	// "regexp"
	// "time"
	// "log"
	// "strconv"
)

func GetParam(r *http.Request, p string, d string) string {

	params, _ := url.ParseQuery(r.URL.RawQuery)
	value := params.Get(p)
	if len(value) > 0 {
		return value
	} else {
		return d
	}
}

func GetAllParams(r *http.Request) map[string][]string {

	params, _ := url.ParseQuery(r.URL.RawQuery)
	return params

}

func GetAllHeaders(r *http.Request) map[string][]string {

	headers := r.Header
	return headers

}

func GetHeader(r *http.Request, p string, d string) string {

	headers := r.Header
	value := headers.Get(p)
	if len(value) > 0 {
		return value
	} else {
		return d
	}
}



func GetEnv(key, d string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return d
	}
	return value
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
