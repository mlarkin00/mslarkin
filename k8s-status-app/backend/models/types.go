package models

type Cluster struct {
	Name      string `json:"name"`
	ProjectID string `json:"projectId"`
	Location  string `json:"location"`
	Status    string `json:"status"`
}

type Workload struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"` // Deployment, Service, etc.
	Status    string `json:"status"`
	Ready     string `json:"ready"`
	UpToDate  string `json:"upToDate"`
	Available string `json:"available"`
	Age       string `json:"age"`
}

// Pod represents a Kubernetes Pod.
type Pod struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Age    string `json:"age"`
}
