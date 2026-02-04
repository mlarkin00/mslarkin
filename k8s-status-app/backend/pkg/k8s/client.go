package k8s

import (
	"context"
	"encoding/base64"
	"fmt"
    "net/http"

	"golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
    gke "k8s-status-backend/pkg/gke"
)

type ClientManager struct {
}

func NewClientManager() *ClientManager {
    return &ClientManager{}
}

// GetClient returns a kubernetes clientset for the given cluster
func (m *ClientManager) GetClient(ctx context.Context, cluster gke.ClusterInfo) (*kubernetes.Clientset, error) {
    // Get credentials (ADC)
    ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
    if err != nil {
        return nil, fmt.Errorf("failed to get google token source: %w", err)
    }

    host := fmt.Sprintf("https://%s", cluster.Endpoint)

    caData, err := base64.StdEncoding.DecodeString(cluster.CaCert)
    if err != nil {
        return nil, fmt.Errorf("failed to decode ca cert: %w", err)
    }

    config := &rest.Config{
        Host: host,
        TLSClientConfig: rest.TLSClientConfig{
            CAData: caData,
        },
        WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
            return &oauth2.Transport{
                Source: ts,
                Base:   rt,
            }
        },
    }

    return kubernetes.NewForConfig(config)
}
