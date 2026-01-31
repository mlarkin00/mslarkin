#!/bin/bash
set -e

# Configuration
NAMESPACE="shop-demo-ns"
GSA_NAME="shop-demo-sa"
PROJECT_ID="mslarkin-ext" # Derived from the requested GSA email
GSA_EMAIL="${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# List of Service Accounts in the namespace to bind
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

echo "Linking Service Accounts in ${NAMESPACE} to ${GSA_EMAIL}..."

for KSA in "${SERVICE_ACCOUNTS[@]}"; do
    echo "Processing ServiceAccount: ${KSA}"

    # 1. Allow Kubernetes Service Account to impersonate the Google Service Account
    gcloud iam service-accounts add-iam-policy-binding "${GSA_EMAIL}" \
        --project="${PROJECT_ID}" \
        --role="roles/iam.workloadIdentityUser" \
        --member="serviceAccount:${PROJECT_ID}.svc.id.goog[${NAMESPACE}/${KSA}]" \
        --condition=None \
        --quiet

    # 2. Annotate the Kubernetes Service Account
    kubectl annotate serviceaccount "${KSA}" \
        --namespace "${NAMESPACE}" \
        "iam.gke.io/gcp-service-account=${GSA_EMAIL}" \
        --overwrite
done

echo "Done! All Service Accounts linked."
