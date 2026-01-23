#!/bin/bash
set -e

PROJECT_ID=${GOOGLE_CLOUD_PROJECT:-mslarkin-ext}
REGION=${GOOGLE_CLOUD_LOCATION:-us-central1}
SERVICE_NAME="failure-mode-agent"
IMAGE_NAME="gcr.io/$PROJECT_ID/$SERVICE_NAME"

echo "Building container image..."
# Copy necessary directories for the agent to function
cp -r ../failure-modes .
cp -r ../baseline .

gcloud builds submit --tag $IMAGE_NAME .

# Cleanup
rm -rf failure-modes baseline

echo "Deploying to Cloud Run..."
gcloud run deploy $SERVICE_NAME \
    --image $IMAGE_NAME \
    --platform managed \
    --region $REGION \
    --project $PROJECT_ID \
    --allow-unauthenticated \
    --service-account agent-sa@mslarkin-ext.iam.gserviceaccount.com \
    --set-env-vars GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=$REGION

echo "Deployment complete. Service URL:"
gcloud run services describe $SERVICE_NAME --platform managed --region $REGION --format 'value(status.url)'
