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
	// run "cloud.google.com/go/run/apiv2"
	// runpb "cloud.google.com/go/run/apiv2/runpb"
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
var projectNum string = "79309377625"
var subscriberServiceName string = "go-pubsub-subscriber"
var subscriberRegion string = "us-central1"

func main() {
	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			publishRate := roundFloat(getPublishRate(ctx, topicId, projectId))
			publishRateQ := roundFloat(getPublishRateQ(ctx, topicId, projectId))
			publishRequestRate := roundFloat(getPublishRequestRate(ctx, topicId, projectId))
			publishRequestRateQ := roundFloat(getPublishRequestRateQ(ctx, topicId, projectId))
			ackRate := roundFloat(getAckRate(ctx, subscriptionId, projectId))
			fmt.Printf("Publish Rate: %v (Query: %v), Publish Request Rate: %v (Query: %v), Ack Rate: %v\n",
				publishRate, publishRateQ, publishRequestRate, publishRequestRateQ, ackRate)
			time.Sleep(10 * time.Second)
		}
	}()

	// Receive output from signalChan.
	sig := <-signalChan
	fmt.Printf("%s signal caught\n", sig)

}

func roundFloat(num float64) float64 { return float64(math.Round(num*100) / 100) }

func getPublishRateQ(ctx context.Context, topicId string, projectId string) float64 {
	monitoringMetric := "pubsub.googleapis.com/topic/message_sizes"

	groupBy := []string{"resource.topic_id"}
	metricFilter := fmt.Sprintf("resource.project_id == '%s'"+
		" && resource.topic_id == '%s'",
		projectId, topicId)

	mqlQuery := fmt.Sprintf("fetch pubsub_topic"+
		"| metric '%s'"+
		"| filter %s"+
		"| align delta(1m)"+
		"| every 1m"+
		"| group_by %s, [value_message_sizes_sum: count(value.message_sizes)]"+
		"| align rate(1m)"+
		"| within 1m, -240s",
		monitoringMetric, metricFilter, groupBy)

	// queryStart := time.Now()
	// fmt.Printf("Query Start: %v\n", queryStart)
	metricData := getMetricQuery(ctx, projectId, mqlQuery)
	// fmt.Println("Metric Length:", len(metricData))

	var metricValues []float64
	for i := range metricData {
		// endTime := metricData[i].TimeInterval.EndTime.AsTime()
		// fmt.Printf("Metric End: %v, Value: %v\n", queryStart.Sub(endTime), metricData[i].GetValues())
		metricValues = append(metricValues, metricData[i].GetValues()[0].GetDoubleValue())
	}
	// fmt.Printf("Metric Values: %v\n", metricValues)
	//Return the latest rate datapoint
	return metricValues[0]
}

func getPublishRate(ctx context.Context, topicId string, projectId string) float64 {
	monitoringMetric := "pubsub.googleapis.com/topic/message_sizes"
	aggregationSeconds := 60
	intervalSeconds := 240
	groupBy := []string{"resource.labels.topic_id"}
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.topic_id =\"%s\"",
		monitoringMetric, topicId)

	metricData := getMetricDelta(ctx,
		metricFilter,
		intervalSeconds,
		aggregationSeconds,
		groupBy,
		projectId)

	// for i := range metricData {
	// 	fmt.Printf("Point: %v\n", metricData[i])
	// 	// fmt.Printf("Value: %v\n", metricData[i].Value.Value)
	// 	fmt.Printf("DistributionValue: %v\n", metricData[i].Value.GetDistributionValue())
	// 	fmt.Printf("Distribution Count: %v\n", metricData[i].Value.GetDistributionValue().Count)
	// }
	messagesPerMinute := float64(metricData[0].Value.GetDistributionValue().Count)

	//Return the latest rate datapoint (converted to seconds)
	return messagesPerMinute/60
}

func getAckRate(ctx context.Context, subscriptionId string, projectId string) float64 {
	monitoringMetric := "pubsub.googleapis.com/subscription/ack_message_count"
	aggregationSeconds := 60
	intervalSeconds := 240
	groupBy := []string{"resource.labels.topic_id"}
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.subscription_id =\"%s\"",
		monitoringMetric, subscriptionId)

	metricData := getMetricRate(ctx,
		metricFilter,
		intervalSeconds,
		aggregationSeconds,
		groupBy,
		projectId)

	//Return the latest rate datapoint
	return float64(metricData[0].GetValue().GetDoubleValue())
}

func getPublishRequestRate(ctx context.Context, topicId string, projectId string) float64 {
	monitoringMetric := "pubsub.googleapis.com/topic/send_request_count"
	aggregationSeconds := 60
	intervalSeconds := 120
	groupBy := []string{"resource.labels.topic_id"}
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.topic_id =\"%s\"",
		monitoringMetric, topicId)

	metricData := getMetricRate(ctx,
		metricFilter,
		intervalSeconds,
		aggregationSeconds,
		groupBy,
		projectId)

	//Return the latest rate datapoint
	return float64(metricData[0].GetValue().GetDoubleValue())
}

func getPublishRequestRateQ(ctx context.Context, topicId string, projectId string) float64 {
	monitoringMetric := "pubsub.googleapis.com/topic/send_request_count"

	groupBy := []string{"resource.topic_id"}
	metricFilter := fmt.Sprintf("resource.project_id == '%s'"+
		" && resource.topic_id == '%s'",
		projectId, topicId)

	mqlQuery := fmt.Sprintf("fetch pubsub_topic"+
		"| metric '%s'"+
		"| filter %s"+
		"| align rate(1m)"+
		"| every 1m"+
		"| group_by %s, [value_send_request_count_aggregate: aggregate(value.send_request_count)]"+
		"| within 1m, -120s",
		monitoringMetric, metricFilter, groupBy)

	// fetch pubsub_topic
	// | metric 'pubsub.googleapis.com/topic/send_request_count'
	// | filter (resource.topic_id == 'pull-test')
	// | align rate(1m)
	// | every 1m
	// | group_by [],
	//     [value_send_request_count_aggregate: aggregate(value.send_request_count)]

	metricData := getMetricQuery(ctx, projectId, mqlQuery)

	var metricValues []float64
	for i := range metricData {
		metricValues = append(metricValues, metricData[i].GetValues()[0].GetDoubleValue())
	}

	//Return the latest rate datapoint
	return metricValues[0]
}

func getMetricQuery(ctx context.Context, projectId string, mqlQuery string) []*monitoringpb.TimeSeriesData_PointData {
	c, err := monitoring.NewQueryClient(ctx)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	req := &monitoringpb.QueryTimeSeriesRequest{
		Name:  "projects/" + projectId,
		Query: mqlQuery,
	}
	it := c.QueryTimeSeries(ctx, req)

	var point []*monitoringpb.TimeSeriesData_PointData
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
		}
		// fmt.Printf("TimeSeriesData: %v\n", resp)
		point = resp.GetPointData()
		// pt := resp.GetPointData()
		// for i := range pt {
		// 	fmt.Printf("End: %v, Value: %v\n", pt[i].TimeInterval.EndTime, pt[i].GetValues())
		// }
	}
	return point
}

func getMetricDelta(ctx context.Context,
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
		// CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_NONE,
		PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_DELTA,
		GroupByFields:      groupBy,
		AlignmentPeriod: &durationpb.Duration{
			Seconds: int64(aggregationSeconds),
		},
	}

	//Configure metric interval
	startTime := time.Now().UTC().Add(time.Second * -time.Duration(intervalSeconds+aggregationSeconds))
	endTime := time.Now().UTC().Add(time.Second * -time.Duration(intervalSeconds))

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
		// SecondaryAggregation: secondaryAggregationStruct,
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

func getMetricRate(ctx context.Context,
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
		PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_RATE,
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
