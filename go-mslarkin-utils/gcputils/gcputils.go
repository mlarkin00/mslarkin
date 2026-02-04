package gcputils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"google.golang.org/api/idtoken"
	"golang.org/x/oauth2/google"

	// "google.golang.org/api/pubsub/v1"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
)

func Helloworld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!\n")
}

func parsePath(resource_path string) string {
	r, _ := regexp.Compile(".*/(.*)$")
	return r.FindStringSubmatch(resource_path)[1]
}

// GetIDToken fetches an OIDC ID token for the given audience.
// It tries to use the metadata server first (when running on GCE/GKE),
// falling back to ADC (Application Default Credentials) if running locally.
func GetIDToken(ctx context.Context, audience string) (string, error) {
	if metadata.OnGCE() {
		// Use metadata server client
		c := metadata.NewClient(nil)
		// Encode audience to be safe
		token, err := c.GetWithContext(ctx, "instance/service-accounts/default/identity?audience="+url.QueryEscape(audience)+"&format=full")
		if err != nil {
			return "", fmt.Errorf("failed to get token from metadata: %w", err)
		}
		return token, nil
	}

	// Local development fallback using ADC
	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err == nil {
		token, err := ts.Token()
		if err == nil {
			return token.AccessToken, nil
		}
	}

	// Fallback to gcloud if available (common for User Credentials)
	// This is useful when running locally with 'gcloud auth login' but without SA key.
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-identity-token", "--audiences="+audience)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("idtoken library failed and gcloud fallback failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetAccessToken fetches an OAuth2 access token with the cloud-platform scope.
func GetAccessToken(ctx context.Context) (string, error) {
	// Use google.FindDefaultCredentials which handles ADC and GKE Metadata automatically.
	// We use the cloud-platform scope.
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return token.AccessToken, nil
}

// GetProjectId tries to get the project ID from:
// 1. google.FindDefaultCredentials (if it has project ID)
// 2. Metadata server
// 3. Environment variable (GOOGLE_CLOUD_PROJECT)
func GetProjectId(ctx context.Context) (string, error) {
	// Try ADC first
	creds, err := google.FindDefaultCredentials(ctx)
	if err == nil && creds.ProjectID != "" {
		return creds.ProjectID, nil
	}

	// Try Metadata
	if metadata.OnGCE() {
		c := metadata.NewClient(nil)
		pid, err := c.ProjectIDWithContext(ctx)
		if err == nil && pid != "" {
			return pid, nil
		}
	}

	// Try Env
	pid := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if pid != "" {
		return pid, nil
	}

	return "", fmt.Errorf("failed to determine project ID")
}

func GetProjectNumber(ctx context.Context) (string, error) {
    if !metadata.OnGCE() {
        return "", fmt.Errorf("not running on GCE/GKE")
    }
    c := metadata.NewClient(nil)
    return c.NumericProjectIDWithContext(ctx)
}

func GetRegion(ctx context.Context) (string, error) {
    if !metadata.OnGCE() {
        return "", fmt.Errorf("not running on GCE/GKE")
    }
    c := metadata.NewClient(nil)
    // There is no direct Region method, usually it is inferred from Zone or explicit metadata
    // "instance/region" return the region URL e.g. projects/123/regions/us-central1
    val, err := c.GetWithContext(ctx, "instance/region")
    if err != nil {
        return "", err
    }
    return parsePath(val), nil
}

func GetInstanceId(ctx context.Context) (string, error) {
     if !metadata.OnGCE() {
        return "", fmt.Errorf("not running on GCE/GKE")
    }
    c := metadata.NewClient(nil)
    return c.InstanceIDWithContext(ctx)
}

func GetRunServiceId() string {
	if os.Getenv("K_SERVICE") == "" {
		return os.Getenv("K_SERVICE")
	} else {
		return os.Getenv("GAE_SERVICE")
	}
}

func GetRunRevisionId() string {
	if os.Getenv("K_REVISION") == "" {
		return os.Getenv("K_REVISION")
	} else {
		return os.Getenv("GAE_VERSION")
	}
}

func GetRunService(ctx context.Context, service string, projectId string, region string) (*runpb.Service, error) {
	c, err := run.NewServicesClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}
	defer c.Close()

	serviceId := "projects/" + projectId + "/locations/" + region + "/services/" + service

	req := &runpb.GetServiceRequest{Name: serviceId}
	return c.GetService(ctx, req)
}

func GetRunServiceUrl(ctx context.Context, service string, projectId string, region string) (string, error) {
	runService, err := GetRunService(ctx, service, projectId, region)
	if err != nil {
		return "", err
	}
	return runService.Uri, nil
}

func GetRunLatestRevision(ctx context.Context, service string, projectId string, region string) (string, error) {
	runService, err := GetRunService(ctx, service, projectId, region)
    if err != nil {
        return "", err
    }
	return parsePath(runService.LatestReadyRevision), nil
}

func GetLastUpdateTs(ctx context.Context, service string, projectId string, region string) (time.Time, error) {
	runService, err := GetRunService(ctx, service, projectId, region)
    if err != nil {
        return time.Time{}, err
    }
	updateTime := runService.UpdateTime.AsTime()
	return updateTime.Local(), nil
}

func GetMetricInterval(intervalSeconds int) *monitoringpb.TimeInterval {
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

func GetRunServiceFilter(monitoringMetric string, service string, region string) string {
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.service_name =\"%s\""+
		" AND resource.labels.location =\"%s\"",
		monitoringMetric, service, region)
	return metricFilter
}

func GetRunRevisionFilter(monitoringMetric string, revision string, region string) string {
	metricFilter := fmt.Sprintf("metric.type=\"%s\""+
		" AND resource.labels.revision_name =\"%s\""+
		" AND resource.labels.location =\"%s\"",
		monitoringMetric, revision, region)
	return metricFilter
}

func GetMetricMean(ctx context.Context, monitoringMetric string,
	resourceFilter string,
	intervalSeconds int,
	aggregationSeconds int,
	groupBy []string,
	projectId string) ([]*monitoringpb.Point, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, err
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
		Name:        "projects/" + projectId,
		Filter:      resourceFilter,
		Interval:    GetMetricInterval(intervalSeconds),
		Aggregation: aggregationStruct,
	}

	// Get the time series data.
	it := client.ListTimeSeries(ctx, req)
	var data []*monitoringpb.Point
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Use resp.
		data = append(data, resp.GetPoints()...)
	}
	return data, nil
}

func GetMetricRate(ctx context.Context, monitoringMetric string,
	resourceFilter string,
	intervalSeconds int,
	aggregationSeconds int,
	groupBy []string,
	projectId string) ([]*monitoringpb.Point, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	aggregationStruct := &monitoringpb.Aggregation{
		CrossSeriesReducer: monitoringpb.Aggregation_REDUCE_SUM,
		PerSeriesAligner:   monitoringpb.Aggregation_ALIGN_RATE,
		GroupByFields:      groupBy,
		AlignmentPeriod: &duration.Duration{
			Seconds: int64(aggregationSeconds),
		},
	}

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:        "projects/" + projectId,
		Filter:      resourceFilter,
		Interval:    GetMetricInterval(intervalSeconds),
		Aggregation: aggregationStruct,
	}

	// Get the time series data.
	it := client.ListTimeSeries(ctx, req)
	var data []*monitoringpb.Point
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		// Use resp.
		data = append(data, resp.GetPoints()...)
	}
	return data, nil
}

func GetRunInstanceCount(ctx context.Context, service string, projectId string, region string) (int, error) {
	monitoringMetric := "run.googleapis.com/container/instance_count"
	aggregationSeconds := 60
	intervalSeconds := 240
	groupBy := []string{"resource.labels.service_name"}

	metricFilter := GetRunServiceFilter(monitoringMetric, service, region)

	metricData, err := GetMetricMean(ctx, monitoringMetric,
		metricFilter,
		intervalSeconds,
		aggregationSeconds,
		groupBy,
		projectId)
    if err != nil {
        return 0, err
    }

    if len(metricData) == 0 {
        return 0, nil
    }

	return int(metricData[0].GetValue().GetDoubleValue()), nil
}
