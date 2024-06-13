package main

import (
	"context"
	"fmt"
	"os"

	// "regexp"
	"math"
	"os/signal"
	// "runtime"
	"syscall"
	"time"

	// "io"
	// "sync/atomic"
	// gcputils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/gcputils"
	// "encoding/json"
	// goutils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/goutils"
	// loadgen "github.com/mlarkin00/mslarkin/go-mslarkin-utils/loadgen"
	// pubsub "cloud.google.com/go/pubsub"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	// expiresIn string `json:"expires_in"`
	// tokenType string `json:"tokenType"`
}

var topicId string = "pull-test"                 //"my-topic"
var subscriptionId string = "pull-queue-testing" //"my-pull-subscription"
var projectId string = "mslarkin-ext"            //"my-project-id"
var subscriberServiceName string = "go-pubsub-subscriber"
var subscriberRegion string = "us-central1"

func main() {
	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for {
			// publishRate := roundFloat(getPublishRate(ctx, topicId, projectId))
			// ackRate := roundFloat(getAckRate(ctx, subscriptionId, projectId))
			// fmt.Printf("Publish Rate: %v, Ack Rate: %v\n", publishRate, ackRate)
			subscriberService, _ := getRunService(ctx, subscriberServiceName, projectId, subscriberRegion)
			configuredInstances := subscriberService.Scaling.MinInstanceCount
			currentInstances := getInstanceCount(ctx, subscriberServiceName, projectId, subscriberRegion)
			fmt.Printf("Service: %s, Instances: Configured: %v | Current: %v\n",
				subscriberServiceName, configuredInstances, currentInstances)
			time.Sleep(10 * time.Second)
		}
	}()

	// var token TokenResponse

	// tokenResp := `{
	// 	"access_token":"ya29.AHES6ZRN3-HlhAPya30GnW_bHSb_QtAS08i85nHq39HE3C2LTrCARA",
	// 	"expires_in":3599,
	// 	"token_type":"Bearer"
	// 	}`

	// json.Unmarshal([]byte(tokenResp), &token)

	// fmt.Println(tokenResp)
	// fmt.Println(token)
	// fmt.Println(token.AccessToken)

	// projectId := "mslarkin-ext"
	// topicId := "pull-test"
	// // subId := "pull-queue-testing"

	// go func() {
	// 	for {
	// 		getPubsubPublishRate(topicId, projectId)
	// 		time.Sleep(10 * time.Second)
	// 	}
	// }()

	// runService, err = gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1")
	// udt := gcputils.GetLastUpdateTs("pubsub-pull-subscriber", "mslarkin-ext", "us-central1")
	// fmt.Println(udt)
	// fmt.Println(gcputils.GetInstanceCount("go-worker", "mslarkin-ext", "us-central1"))
	// fmt.Println(gcputils.GetRunService("go-worker", "mslarkin-ext", "us-central1"))

	// Receive output from signalChan.
	sig := <-signalChan
	fmt.Printf("%s signal caught\n", sig)

}

func roundFloat(num float64) float64 { return float64(math.Round(num*100) / 100) }

func getInstanceCount(ctx context.Context, service string, projectId string, region string) int {
	monitoringMetric := "run.googleapis.com/container/instance_count"
	aggregationSeconds := 60
	intervalSeconds := 240
	groupBy := []string{"resource.labels.service_name"}
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.service_name =\"%s\""+
		" AND resource.labels.location =\"%s\"",
		monitoringMetric, service, region)

	metricData := getMetricMean(ctx,
		metricFilter,
		intervalSeconds,
		aggregationSeconds,
		groupBy,
		projectId)

	fmt.Println(metricData)
	return int(metricData[0].GetValue().GetDoubleValue())
}

func getMetricMean(ctx context.Context,
	resourceFilter string,
	intervalSeconds int,
	aggregationSeconds int,
	groupBy []string,
	projectId string) []monitoringpb.Point {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	//Configure metric aggregation
	aggregationStruct := &monitoringpb.Aggregation{
		CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
		PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_MEAN,
		GroupByFields:      groupBy,
		AlignmentPeriod: &durationpb.Duration{
			Seconds: int64(aggregationSeconds),
		},
	}

	//Configure metric interval
	startTime := time.Now().UTC().Add(time.Second * -time.Duration(intervalSeconds))
	endTime := time.Now().UTC()

	interval := &monitoringpb.TimeInterval{
		StartTime: &timestamppb.Timestamp{
			Seconds: startTime.Unix(),
		},
		EndTime: &timestamppb.Timestamp{
			Seconds: endTime.Unix(),
		},
	}

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:        "projects/" + projectId,
		Filter:      resourceFilter,
		Interval:    interval,
		Aggregation: aggregationStruct,
	}

	// Get the time series data.
	it := client.ListTimeSeries(ctx, req)
	var data []monitoringpb.Point
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Handle error.
			panic(err)
		}
		// Use resp.
		for _, point := range resp.GetPoints() {
			data = append(data, *point)
		}
	}
	return data
}

func getRunService(ctx context.Context,
	service string,
	projectId string,
	region string) (*runpb.Service, error) {
	client, err := run.NewServicesClient(ctx)
	if err != nil {
		fmt.Printf("Error getting client:\n")
		panic(err)
	}
	defer client.Close()

	serviceId := "projects/" + projectId + "/locations/" + region + "/services/" + service

	req := &runpb.GetServiceRequest{Name: serviceId}
	resp, err := client.GetService(ctx, req)
	if err != nil {
		fmt.Printf("Error getting service: %s in %s (Project ID: %s)\n", service, region, projectId)
		panic(err)
	}
	return resp, err

}