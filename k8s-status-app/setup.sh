#!/bin/bash
set -e

# Configuration
PROJECT_ID="mslarkin-ext"
CLUSTER="ai-auto-cluster"
LOCATION="us-west1"
BACKEND_SA="k8s-status-backend-sa"
FRONTEND_SA="k8s-status-frontend-sa"

echo "Setting up resources for GKE Status App..."

# Create Google Service Accounts
if ! gcloud iam service-accounts describe ${BACKEND_SA}@${PROJECT_ID}.iam.gserviceaccount.com > /dev/null 2>&1; then
    echo "Creating Backend Service Account..."
    gcloud iam service-accounts create ${BACKEND_SA} --display-name="GKE Status Backend SA"
fi

if ! gcloud iam service-accounts describe ${FRONTEND_SA}@${PROJECT_ID}.iam.gserviceaccount.com > /dev/null 2>&1; then
    echo "Creating Frontend Service Account..."
    gcloud iam service-accounts create ${FRONTEND_SA} --display-name="GKE Status Frontend SA"
fi

# Bind Workload Identity
echo "Binding Workload Identity..."
gcloud iam service-accounts add-iam-policy-binding ${BACKEND_SA}@${PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${PROJECT_ID}.svc.id.goog[default/${BACKEND_SA}]"

gcloud iam service-accounts add-iam-policy-binding ${FRONTEND_SA}@${PROJECT_ID}.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:${PROJECT_ID}.svc.id.goog[default/${FRONTEND_SA}]"

# Grant permissions to Backend SA
# Note: Specific roles for MCP access would be needed here.
# Assuming 'viewer' for demo purposes, but in reality, it needs access to GKE API and specific endpoints.
echo "Granting permissions..."
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
    --member "serviceAccount:${BACKEND_SA}@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role "roles/container.viewer"

# Apply Kubernetes Manifests
echo "Applying manifests..."
# Assuming kubectl is configured for the correct cluster
kubectl apply -f deploy/backend.yaml
kubectl apply -f deploy/frontend.yaml

echo "Setup complete."
