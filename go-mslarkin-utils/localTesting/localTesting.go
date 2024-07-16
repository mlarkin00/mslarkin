package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

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
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"

	// pubsub "cloud.google.com/go/pubsub"
	// run "cloud.google.com/go/run/apiv2"
	// runpb "cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)
var lastUpdate time.Time

var topicId string = "pull-test"                 //"my-topic"
var subscriptionId string = "pull-queue-testing" //"my-pull-subscription"
var projectId string = "mslarkin-ext"            //"my-project-id"
// var projectNum string = "79309377625"
var subscriberServiceName string = "go-pubsub-subscriber"
var subscriberRegion string = "us-central1"

var targetMaxAgeS float64 = 5
var targetAckLatencyMs float64 = 800
var jitterRange = .1   // Allow some range of values without action
var updateDelayMin = 5 // Time to wait, after a change, before making any other changes

func main() {
	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Getenv("JITTER_RANGE")) > 0 {
		jitterRange, _ = strconv.ParseFloat(os.Getenv("JITTER_RANGE"), 64)
	}

	go func() {
		for {
			// publishRate := roundFloat(getPublishRate(ctx, topicId, projectId))
			// ackRate := roundFloat(getAckRate(ctx, subscriptionId, projectId))
			// pubAckDeltaRate := publishRate - ackRate
			// fmt.Printf("Publish Rate: %v, Ack Rate: %v (Delta: %v)\n", publishRate, ackRate, pubAckDeltaRate)

			// ackLatency := roundFloat(getAckLatencyMs(ctx, subscriptionId, projectId))
			// fmt.Printf("Ack Latency: %v\n", ackLatency)

			// messageBacklog := getMessageBacklog(ctx, subscriptionId, projectId)
			// maxMessageAge := getMaxMessageAgeS(ctx, subscriptionId, projectId)
			// fmt.Printf("Backlog: %v, Max Age: %v\n", messageBacklog, maxMessageAge)

			// subscriberService, _ := getRunService(ctx, subscriberServiceName, projectId, subscriberRegion)
			// configuredInstances := subscriberService.Scaling.MinInstanceCount
			// currentInstances := getInstanceCount(ctx, subscriberServiceName, projectId, subscriberRegion)
			// lastUpdate = subscriberService.UpdateTime.AsTime()
			// fmt.Printf("Service: %s, Instances: Configured: %v | Current: %v (Last Updated: %v)\n",
			// 	subscriberServiceName, configuredInstances, currentInstances, lastUpdate)

			// // Autoscaling logic
			// go scalingCheck(ctx)

			monitoringMetric := "prometheus.googleapis.com/sidecar_otel_counter_total/counter"
			metricFilter := fmt.Sprintf("metric.type=\"%s\"", monitoringMetric)
			groupBy := []string{} //[]string{"resource.labels.service_name"}
			data := getMetricRate(ctx,
				metricFilter,
				10,
				60,
				groupBy,
				projectId)

			fmt.Printf("Metric:\n%v\n----\n", data)
			time.Sleep(30 * time.Second)
		}
	}()

	// Receive output from signalChan.
	sig := <-signalChan
	fmt.Printf("%s signal caught\n", sig)

}

func roundFloat(num float64) float64 { return float64(math.Round(num*100) / 100) }

/////////////////
// Autoscaling //
/////////////////

// func scalingCheck(ctx context.Context) {
// 	var currentInstances int
// 	var configuredInstances int32
// 	var recommendedInstances int32
// 	var subscriberService *runpb.Service

// 	// Check delta between publish and ack rate
// 	publishRate := roundFloat(getPublishRate(ctx, topicId, projectId))
// 	ackRate := roundFloat(getAckRate(ctx, subscriptionId, projectId))
// 	pubAckDeltaRate := publishRate - ackRate
// 	if math.Abs(pubAckDeltaRate) >= (publishRate * jitterRange) {
// 		fmt.Printf("Pub / Ack Delta (messages/s): %v (Jitter (%v): +/-%v)\n",
// 			pubAckDeltaRate,
// 			jitterRange,
// 			(publishRate * jitterRange))
// 	}

// 	ackLatencyMs := roundFloat(getAckLatencyMs(ctx, subscriptionId, projectId))
// 	fmt.Printf("Ack Latency (ms): %v\n", ackLatencyMs)
// 	if math.Abs(targetAckLatencyMs-ackLatencyMs) > (targetAckLatencyMs * jitterRange) {
// 		fmt.Printf("Ack Latency Delta: %v, Range: +/-%v\n", math.Abs(targetAckLatencyMs-ackLatencyMs), (targetAckLatencyMs * jitterRange))
// 		currentInstances = getInstanceCount(ctx, subscriberServiceName, projectId, subscriberRegion)
// 		recommendedInstances = averageValueRecommendation(ackLatencyMs, targetAckLatencyMs, currentInstances)
// 		fmt.Printf("Recommended Instance change (Ack Latency): %v --> %v\n", currentInstances, recommendedInstances)
// 	}

// 	// messageBacklog := getMessageBacklog(ctx, subscriptionId, projectId)
// 	// fmt.Printf("Backlog (messages): %v\n", messageBacklog)

// 	maxMessageAgeS := getMaxMessageAgeS(ctx, subscriptionId, projectId)
// 	// fmt.Printf("Max Age (s): %v\n", maxMessageAgeS)
// 	// Only using Max Age to scale up
// 	if (maxMessageAgeS - targetMaxAgeS) > (targetMaxAgeS * jitterRange) {
// 		fmt.Printf("Max Age Delta: %v, Range: +/-%v\n", math.Abs(targetMaxAgeS-maxMessageAgeS), (targetMaxAgeS * jitterRange))
// 		// currentInstances := getInstanceCount(ctx, subscriberServiceName, projectId, subscriberRegion)
// 		// recommendedInstances := averageValueRecommendation(maxMessageAgeS, targetMaxAgeS, currentInstances)
// 		// fmt.Printf("Recommended Instance change (Age): %v --> %v\n", currentInstances, recommendedInstances)

// 		subscriberService, _ = getRunService(ctx, subscriberServiceName, projectId, subscriberRegion)
// 		configuredInstances = subscriberService.Scaling.MinInstanceCount
// 		// fmt.Printf("Configured: %v | Current: %v\n", configuredInstances, currentInstances)
// 	}

// 	// If the recommendation is different than the current configuration, and the change delay has expired
// 	if configuredInstances != recommendedInstances &&
// 		time.Since(subscriberService.UpdateTime.AsTime()) > (time.Duration(updateDelayMin)*time.Minute) {
// 		fmt.Printf("Time since last change: %v\n", time.Since(subscriberService.UpdateTime.AsTime()))
// 		// fmt.Println("Updating Instances...")
// 		// updateInstanceCount(ctx, subscriberService, recommendedInstances)
// 		// fmt.Println("Instance count updated")
// 	}

// }

// func averageValueRecommendation(metricValue float64, metricTarget float64, currentInstanceCount int) int32 {
// 	recommendedInstances := math.Ceil(float64(currentInstanceCount) * (metricValue / metricTarget))
// 	return int32(recommendedInstances)
// }

// func targetValueRecommendation(metricValue float64, metricTarget float64, currentInstanceCount int) int32 {
// 	recommendedInstances := math.Ceil(float64(currentInstanceCount) * (metricTarget / metricValue))
// 	return int32(recommendedInstances)
// }

// func instanceCapacityRecommendation(metricValue float64, instanceCapacity float64) int32 {
// 	recommendedInstances := math.Ceil(metricValue / instanceCapacity)
// 	return int32(recommendedInstances)
// }

// //////////////////////////
// // Cloud Run management //
// //////////////////////////

// func getInstanceCount(ctx context.Context, service string, projectId string, region string) int {
// 	monitoringMetric := "run.googleapis.com/container/instance_count"
// 	aggregationSeconds := 60
// 	metricDelaySeconds := 180
// 	groupBy := []string{"resource.labels.service_name"}
// 	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
// 		" AND resource.labels.service_name =\"%s\""+
// 		" AND resource.labels.location =\"%s\"",
// 		monitoringMetric, service, region)

// 	metricData := getMetricMean(ctx,
// 		metricFilter,
// 		metricDelaySeconds,
// 		aggregationSeconds,
// 		groupBy,
// 		projectId)

// 	// fmt.Printf("Instance Metric Delay: %v\n", time.Since(metricData[0].GetInterval().EndTime.AsTime()))

// 	return int(metricData[0].GetValue().GetDoubleValue())
// }

// func getRunService(ctx context.Context,
// 	service string,
// 	projectId string,
// 	region string) (*runpb.Service, error) {
// 	client, err := run.NewServicesClient(ctx)
// 	if err != nil {
// 		fmt.Printf("Error getting client:\n")
// 		panic(err)
// 	}
// 	defer client.Close()

// 	serviceId := "projects/" + projectId + "/locations/" + region + "/services/" + service

// 	req := &runpb.GetServiceRequest{Name: serviceId}
// 	resp, err := client.GetService(ctx, req)
// 	if err != nil {
// 		fmt.Printf("Error getting service: %s in %s (Project ID: %s)\n", service, region, projectId)
// 		panic(err)
// 	}
// 	return resp, err

// }

// func updateInstanceCount(ctx context.Context, service *runpb.Service, desiredMinInstances int32) {
// 	// Return if the current min-instances setting is the same as the desired min-instances
// 	if service.Scaling.MinInstanceCount == int32(desiredMinInstances) {
// 		return
// 	}

// 	client, err := run.NewServicesClient(ctx)
// 	if err != nil {
// 		fmt.Printf("Error getting client:\n")
// 		panic(err)
// 	}
// 	defer client.Close()

// 	service.Scaling.MinInstanceCount = desiredMinInstances

// 	req := &runpb.UpdateServiceRequest{
// 		Service: service,
// 	}
// 	op, err := client.UpdateService(ctx, req)
// 	if err != nil {
// 		fmt.Printf("Error in update service request:\n")
// 		panic(err)
// 	}

// 	resp, err := op.Wait(ctx)
// 	if err != nil {
// 		fmt.Printf("Error updating service:\n")
// 		panic(err)
// 	}
// 	_ = resp
// }

//////////////////////////////
// Cloud Monitoring metrics //
//////////////////////////////

// pubsub.googleapis.com/subscription/sent_message_count (delay: 240s)
// pubsub.googleapis.com/subscription/delivery_latency_health_score (delay: 360s)

// func getMessageBacklog(ctx context.Context, subscriptionId string, projectId string) int {
// 	monitoringMetric := "pubsub.googleapis.com/subscription/num_undelivered_messages"
// 	aggregationSeconds := 60
// 	metricDelaySeconds := 120
// 	groupBy := []string{"resource.labels.subscription_id"}
// 	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
// 		" AND resource.labels.subscription_id =\"%s\"",
// 		monitoringMetric, subscriptionId)

// 	metricData := getMetricMean(ctx,
// 		metricFilter,
// 		metricDelaySeconds,
// 		aggregationSeconds,
// 		groupBy,
// 		projectId)

// 	return int(metricData[0].GetValue().GetDoubleValue())
// }

// func getMaxMessageAgeS(ctx context.Context, subscriptionId string, projectId string) float64 {
// 	monitoringMetric := "pubsub.googleapis.com/subscription/oldest_unacked_message_age"
// 	aggregationSeconds := 60
// 	metricDelaySeconds := 120
// 	groupBy := []string{"resource.labels.subscription_id"}
// 	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
// 		" AND resource.labels.subscription_id =\"%s\"",
// 		monitoringMetric, subscriptionId)

// 	metricData := getMetricMean(ctx,
// 		metricFilter,
// 		metricDelaySeconds,
// 		aggregationSeconds,
// 		groupBy,
// 		projectId)

// 	return metricData[0].GetValue().GetDoubleValue()
// }

// func getPublishRate(ctx context.Context, topicId string, projectId string) float64 {
// 	monitoringMetric := "pubsub.googleapis.com/topic/message_sizes"
// 	aggregationSeconds := 60
// 	metricDelaySeconds := 240
// 	groupBy := []string{"resource.labels.topic_id"}
// 	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
// 		" AND resource.labels.topic_id =\"%s\"",
// 		monitoringMetric, topicId)

// 	metricData := getMetricDelta(ctx,
// 		metricFilter,
// 		metricDelaySeconds,
// 		aggregationSeconds,
// 		groupBy,
// 		projectId)

// 	messagesPerMinute := float64(metricData[0].Value.GetDistributionValue().Count)

// 	//Return the latest rate datapoint (converted to seconds)
// 	return messagesPerMinute / 60
// }

// func getAckRate(ctx context.Context, subscriptionId string, projectId string) float64 {
// 	monitoringMetric := "pubsub.googleapis.com/subscription/ack_message_count"
// 	aggregationSeconds := 60
// 	metricDelaySeconds := 240
// 	groupBy := []string{"resource.labels.subscription_id"}
// 	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
// 		" AND resource.labels.subscription_id =\"%s\"",
// 		monitoringMetric, subscriptionId)

// 	metricData := getMetricRate(ctx,
// 		metricFilter,
// 		metricDelaySeconds,
// 		aggregationSeconds,
// 		groupBy,
// 		projectId)

// 	//Return the latest rate datapoint
// 	return float64(metricData[0].GetValue().GetDoubleValue())
// }

// func getAckLatencyMs(ctx context.Context, subscriptionId string, projectId string) float64 {
// 	monitoringMetric := "pubsub.googleapis.com/subscription/ack_latencies"
// 	aggregationSeconds := 60
// 	metricDelaySeconds := 240
// 	groupBy := []string{"resource.labels.subscription_id"}
// 	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
// 		" AND resource.labels.subscription_id =\"%s\"",
// 		monitoringMetric, subscriptionId)

// 	//TODO: Implement Distribution metric mean function
// 	metricData := getMetricDelta(ctx,
// 		metricFilter,
// 		metricDelaySeconds,
// 		aggregationSeconds,
// 		groupBy,
// 		projectId)

// 	//Return the latest rate datapoint
// 	return float64(metricData[0].GetValue().GetDistributionValue().Mean)
// }

func getMetricRate(ctx context.Context,
	resourceFilter string,
	metricDelaySeconds int,
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
		PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_RATE,
		GroupByFields:      groupBy,
		AlignmentPeriod: &durationpb.Duration{
			Seconds: int64(aggregationSeconds),
		},
	}

	//Configure metric interval
	startTime := time.Now().UTC().Add(time.Second * -time.Duration(metricDelaySeconds+aggregationSeconds))
	endTime := time.Now().UTC() //.Add(time.Second * -time.Duration(metricDelaySeconds))

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
		fmt.Printf("Response:\n%v\n", resp)
		for _, point := range resp.GetPoints() {
			data = append(data, *point)
		}
	}
	return data
}

func getMetricMean(ctx context.Context,
	resourceFilter string,
	metricDelaySeconds int,
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

	///Configure metric interval
	startTime := time.Now().UTC().Add(time.Second * -time.Duration(metricDelaySeconds+aggregationSeconds))
	endTime := time.Now().UTC() //.Add(time.Second * -time.Duration(metricDelaySeconds))

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

func getMetricDelta(ctx context.Context,
	resourceFilter string,
	metricDelaySeconds int,
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
		PerSeriesAligner: monitoringpb.Aggregation_ALIGN_DELTA,
		GroupByFields:    groupBy,
		AlignmentPeriod: &durationpb.Duration{
			Seconds: int64(aggregationSeconds),
		},
	}

	//Configure metric interval
	startTime := time.Now().UTC().Add(time.Second * -time.Duration(metricDelaySeconds+aggregationSeconds))
	endTime := time.Now().UTC() //.Add(time.Second * -time.Duration(metricDelaySeconds))

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
