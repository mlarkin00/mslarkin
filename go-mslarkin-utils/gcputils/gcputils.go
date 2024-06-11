package gcputils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	// "google.golang.org/api/pubsub/v1"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/api/iterator"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func Helloworld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!\n")
}

func parsePath(resource_path string) string {
	r, _ := regexp.Compile(".*/(.*)$")
	return r.FindStringSubmatch(resource_path)[1]
}

func GetProjectId() string {
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		bodyString := string(bodyBytes)
		return bodyString
	} else {
		return string(resp.StatusCode)
	}

}

func GetProjectNumber() string {
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		bodyString := string(bodyBytes)
		return bodyString
	} else {
		return string(resp.StatusCode)
	}

}

func GetRegion() string {
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/region", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		bodyString := string(bodyBytes)
		return parsePath(bodyString)
	} else {
		return string(resp.StatusCode)
	}

}

func GetInstanceId() string {
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/id", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Metadata-Flavor", "Google")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		bodyString := string(bodyBytes)
		return bodyString
	} else {
		return string(resp.StatusCode)
	}

}

func GetServiceId() string {
	if os.Getenv("K_SERVICE") == "" {
		return os.Getenv("K_SERVICE")
	} else {
		return os.Getenv("GAE_SERVICE")
	}
}

func GetRevisionId() string {
	if os.Getenv("K_REVISION") == "" {
		return os.Getenv("K_REVISION")
	} else {
		return os.Getenv("GAE_VERSION")
	}
}

func GetRunService(service string, projectId string, region string) (*runpb.Service, error) {
	ctx := context.Background()
	c, err := run.NewServicesClient(ctx)
	if err != nil {
		fmt.Printf("Error getting client:\n")
		panic(err)
	}
	defer c.Close()

	serviceId := "projects/" + projectId + "/locations/" + region + "/services/" + service

	req := &runpb.GetServiceRequest{Name: serviceId}
	resp, err := c.GetService(ctx, req)
	if err != nil {
		fmt.Printf("Error getting service:\n")
		panic(err)
	}
	return resp, err

}

func GetServiceUrl(service string, projectId string, region string) string {
	runService, err := GetRunService("pubsub-pull-subscriber", "mslarkin-ext", "us-central1")
	if err != nil {
		fmt.Printf("Error getting service:\n")
		panic(err)
	}
	return runService.Uri
}

func GetLatestRevision(service string, projectId string, region string) string {
	runService, _ := GetRunService(service, projectId, region)
	return parsePath(runService.LatestReadyRevision)
}

func GetLastUpdateTs(service string, projectId string, region string) time.Time {
	runService, _ := GetRunService(service, projectId, region)
	updateTime := runService.UpdateTime.AsTime()
	return updateTime.Local()
}

func getMetricInterval(intervalSeconds int) *monitoringpb.TimeInterval {
	startTime := time.Now().UTC().Add(time.Second * -time.Duration(intervalSeconds))
	endTime := time.Now().UTC()

	return &monitoringpb.TimeInterval{
		StartTime: &timestamp.Timestamp{
				Seconds: startTime.Unix(),
			},
		EndTime: &timestamp.Timestamp{
				Seconds: endTime.Unix(),
			},
		}
}

func GetMetricMean(monitoringMetric string, 
				resourceFilter string, 
				intervalSeconds int,
				aggregationSeconds int,
				groupBy []string,
				projectId string) []monitoringpb.Point {
	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	aggregationStruct := &monitoringpb.Aggregation{
		CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
		PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_MEAN,
		GroupByFields:      groupBy,
		AlignmentPeriod: &duration.Duration{
			Seconds: int64(aggregationSeconds),
		},
	}

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectId,
		Filter: resourceFilter,
		Interval: getMetricInterval(intervalSeconds),
		Aggregation: aggregationStruct,
	}

	// fmt.Println("Metric:", monitoringMetric)
	// fmt.Println("Filter:", resourceFilter)
	// fmt.Println("Interval:", intervalSeconds, "Agg:",aggregationSeconds)
	// fmt.Println("groupBy", groupBy)
	// fmt.Println("project", projectId)

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

func GetInstanceCount(service string, projectId string, region string) int {
	monitoringMetric := "run.googleapis.com/container/instance_count"
	aggregationSeconds := 60
	intervalSeconds := 240
	groupBy := []string{"resource.labels.service_name"}

	metricFilter := "metric.type=\"" + monitoringMetric +
					 "\" AND resource.labels.service_name =\"" +
					  service + 
					  "\" AND resource.labels.location =\"" +
					  region + "\""

	metricData := GetMetricMean(monitoringMetric,
					metricFilter,
					intervalSeconds,
					aggregationSeconds,
					groupBy,
					projectId)

	return int(metricData[0].GetValue().GetDoubleValue())
}
