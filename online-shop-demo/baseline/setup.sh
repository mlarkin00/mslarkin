#!/bin/bash
set -e

# Configuration
REGION="${REGION:-us-west1}"
PROJECT_ID="${PROJECT_ID:-mslarkin-ext}"
NAMESPACE="${NAMESPACE:-shop-demo-ns}"
CLUSTER_NAME="${CLUSTER_NAME:-onlineboutique-cluster}" # AlloyDB Cluster
INSTANCE_NAME="${INSTANCE_NAME:-onlineboutique-instance}" # AlloyDB Instance
ALLOYDB_NETWORK="${ALLOYDB_NETWORK:-shop-demo-network}"
ALLOYDB_SERVICE_NAME="${ALLOYDB_SERVICE_NAME:-onlineboutique-network-range}"

# Service Accounts
GSA_NAME="alloydb-user-sa" # Using the one we already set up
GSA_EMAIL="${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Secrets
GCP_SECRET_NAME="ai-studio-api-key"
K8S_SECRET_NAME="ai-studio-api-key"
ALLOYDB_SECRET_NAME="alloydb-secret"

# 1. Enable APIs
echo "Enabling necessary Google Cloud APIs..."
gcloud services enable \
    monitoring.googleapis.com \
    cloudtrace.googleapis.com \
    cloudprofiler.googleapis.com \
    alloydb.googleapis.com \
    servicenetworking.googleapis.com \
    secretmanager.googleapis.com \
    aiplatform.googleapis.com \
    --project "${PROJECT_ID}"

# 2. IAM & Workload Identity
echo "Configuring IAM & Workload Identity..."

# Create GSA if it doesn't exist
if ! gcloud iam service-accounts describe "${GSA_EMAIL}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    echo "Creating Service Account ${GSA_NAME}..."
    gcloud iam service-accounts create "${GSA_NAME}" --display-name="${GSA_NAME}" --project="${PROJECT_ID}" || true
fi

# Grant Roles to GSA
echo "Granting roles to ${GSA_EMAIL}..."
ROLES=(
    "roles/cloudtrace.agent"
    "roles/monitoring.metricWriter"
    "roles/cloudprofiler.agent"
    "roles/alloydb.client"
    "roles/secretmanager.secretAccessor"
    "roles/aiplatform.user"
)

for ROLE in "${ROLES[@]}"; do
    gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
        --member="serviceAccount:${GSA_EMAIL}" \
        --role="${ROLE}" \
        --condition=None \
        --quiet >/dev/null
done

# Bind KSA to GSA (Workload Identity)
SERVICE_ACCOUNTS=(
    "adservice"
    "cartservice"
    "checkoutservice"
    "currencyservice"
    "emailservice"
    "frontend"
    "loadgenerator"
    "paymentservice"
    "productcatalogservice"
    "recommendationservice"
    "shippingservice"
    "shoppingassistantservice"
)

echo "Binding Kubernetes Service Accounts in ${NAMESPACE}..."
for KSA in "${SERVICE_ACCOUNTS[@]}"; do
    # Bind
    gcloud iam service-accounts add-iam-policy-binding "${GSA_EMAIL}" \
        --project="${PROJECT_ID}" \
        --role="roles/iam.workloadIdentityUser" \
        --member="serviceAccount:${PROJECT_ID}.svc.id.goog[${NAMESPACE}/${KSA}]" \
        --condition=None \
        --quiet >/dev/null

    # Annotate (Clean way: only if it exists in cluster to avoid errors if not deployed yet?)
    # But user wants setup.sh to be runnable. We'll assume kubectl context is set.
    if kubectl get serviceaccount "${KSA}" -n "${NAMESPACE}" >/dev/null 2>&1; then
        kubectl annotate serviceaccount "${KSA}" \
            --namespace "${NAMESPACE}" \
            "iam.gke.io/gcp-service-account=${GSA_EMAIL}" \
            --overwrite
    else
        echo "Warning: KSA ${KSA} not found in namespace ${NAMESPACE}. Skipping annotation (binding still created)."
    fi
done

# 3. AlloyDB Infrastructure
echo "Provisioning AlloyDB Infrastructure..."

# Create Secret for DB Password if not exists
if ! gcloud secrets describe "${ALLOYDB_SECRET_NAME}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    echo "Creating secret ${ALLOYDB_SECRET_NAME} with random password..."
    DB_PASSWORD=$(openssl rand -base64 12)
    echo -n "${DB_PASSWORD}" | gcloud secrets create "${ALLOYDB_SECRET_NAME}" --data-file=- --project="${PROJECT_ID}"
else
    echo "Secret ${ALLOYDB_SECRET_NAME} already exists."
fi

# VPC Peering (Service Networking)
if ! gcloud compute addresses describe "${ALLOYDB_SERVICE_NAME}" --global --project="${PROJECT_ID}" >/dev/null 2>&1; then
    echo "Creating reserved IP range for Private Services Access..."
    gcloud compute addresses create "${ALLOYDB_SERVICE_NAME}" \
        --global \
        --purpose=VPC_PEERING \
        --prefix-length=16 \
        --description="Online Boutique Private Services" \
        --network="${ALLOYDB_NETWORK}" \
        --project="${PROJECT_ID}"
fi

# Connect VPC Peering
# This command is idempotent but can fail if already peered with different config. We assume valid state or manual fix if conflict.
echo "Ensuring VPC Peering..."
gcloud services vpc-peerings connect \
    --service=servicenetworking.googleapis.com \
    --ranges="${ALLOYDB_SERVICE_NAME}" \
    --network="${ALLOYDB_NETWORK}" \
    --project="${PROJECT_ID}" || echo "VPC peering might already exist or require checking."

# Create AlloyDB Cluster
if ! gcloud alloydb clusters describe "${CLUSTER_NAME}" --region="${REGION}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    echo "Creating AlloyDB Cluster ${CLUSTER_NAME}..."
    # Fetch password from secret
    DB_PASSWORD=$(gcloud secrets versions access latest --secret="${ALLOYDB_SECRET_NAME}" --project="${PROJECT_ID}")
    gcloud alloydb clusters create "${CLUSTER_NAME}" \
        --region="${REGION}" \
        --password="${DB_PASSWORD}" \
        --disable-automated-backup \
        --network="${ALLOYDB_NETWORK}" \
        --project="${PROJECT_ID}"
else
    echo "AlloyDB Cluster ${CLUSTER_NAME} already exists."
fi

# Create AlloyDB Instance (Primary)
if ! gcloud alloydb instances describe "${INSTANCE_NAME}" --cluster="${CLUSTER_NAME}" --region="${REGION}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    echo "Creating AlloyDB Primary Instance ${INSTANCE_NAME}..."
    gcloud alloydb instances create "${INSTANCE_NAME}" \
        --cluster="${CLUSTER_NAME}" \
        --region="${REGION}" \
        --cpu-count=4 \
        --instance-type=PRIMARY \
        --project="${PROJECT_ID}"
else
    echo "AlloyDB Instance ${INSTANCE_NAME} already exists."
fi

# 4. Sync Secrets to K8s
echo "Syncing Secrets to Kubernetes..."
# AI Studio Key
if gcloud secrets describe "${GCP_SECRET_NAME}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    PAYLOAD=$(gcloud secrets versions access latest --secret="${GCP_SECRET_NAME}" --project="${PROJECT_ID}")
    kubectl create secret generic "${K8S_SECRET_NAME}" \
        --namespace="${NAMESPACE}" \
        --from-literal=latest="${PAYLOAD}" \
        --dry-run=client -o yaml | kubectl apply -f -
    echo "Synced ${K8S_SECRET_NAME}."
else
    echo "Warning: GCP Secret ${GCP_SECRET_NAME} not found. Skipping K8s secret sync."
fi

# AlloyDB Secrets (DB Password)
DB_PASSWORD=$(gcloud secrets versions access latest --secret="${ALLOYDB_SECRET_NAME}" --project="${PROJECT_ID}")
kubectl create secret generic "${ALLOYDB_SECRET_NAME}" \
    --namespace="${NAMESPACE}" \
    --from-literal=password="${DB_PASSWORD}" \
    --dry-run=client -o yaml | kubectl apply -f -
echo "Synced ${ALLOYDB_SECRET_NAME}."

echo "==================================================="
echo "Setup Complete!"
echo "Next Step: Run 'setup-db.sh' in an environment with VPC access to initialized the database."
echo "==================================================="
