#!/usr/bin/env bash
set -euo pipefail

# Deploy a Cloud Run service with OAuth (app-managed auth)
#
# Usage:
#   ./deploy-oauth.sh <service-name> [--region <region>]

REGION="us-central1"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <service-name> [--region <region>]"
  exit 1
fi

SERVICE_NAME="$1"
shift

while [[ $# -gt 0 ]]; do
  case "$1" in
    --region) REGION="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

PROJECT_ID=$(gcloud config get-value project 2>/dev/null)

echo "=== Deploying $SERVICE_NAME to Cloud Run with OAuth ==="
echo "  Project:  $PROJECT_ID"
echo "  Region:   $REGION"
echo ""

# --- Step 1: Enable required APIs ---
echo ">>> Step 1: Enabling required APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com

# --- Step 2: Create OAuth credentials ---
echo ""
echo ">>> Step 2: OAuth credentials setup"
echo ""
echo "  You must create OAuth 2.0 credentials in the Google Cloud Console:"
echo "  1. Go to: https://console.cloud.google.com/apis/credentials"
echo "  2. Click 'Create Credentials' → 'OAuth client ID'"
echo "  3. Application type: 'Web application'"
echo "  4. Add authorized redirect URI (will be shown after deploy)"
echo "  5. Copy the Client ID and Client Secret"
echo ""
read -rp "  Enter GOOGLE_CLIENT_ID: " CLIENT_ID
read -rp "  Enter GOOGLE_CLIENT_SECRET: " CLIENT_SECRET

if [[ -z "$CLIENT_ID" || -z "$CLIENT_SECRET" ]]; then
  echo "Error: Both CLIENT_ID and CLIENT_SECRET are required"
  exit 1
fi

# --- Step 3: Deploy to Cloud Run ---
# KEY TEACHING MOMENT: With OAuth, the app handles authentication itself.
# We use --allow-unauthenticated because users need to reach the login page.
# This is the OPPOSITE of IAP, where --allow-unauthenticated is a security mistake.
echo ">>> Step 3: Deploying to Cloud Run (--allow-unauthenticated)..."
echo "  NOTE: Unlike IAP, OAuth apps MUST allow unauthenticated access"
echo "  because the app itself handles the login flow."
gcloud run deploy "$SERVICE_NAME" \
  --source . \
  --region="$REGION" \
  --allow-unauthenticated \
  --port=8080 \
  --cpu=1 \
  --memory=256Mi \
  --min-instances=0 \
  --max-instances=3 \
  --concurrency=80

# --- Step 4: Get service URL and set env vars ---
SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" --region="$REGION" --format='value(status.url)')
REDIRECT_URL="${SERVICE_URL}/auth/callback"

echo ">>> Step 4: Setting environment variables..."
gcloud run services update "$SERVICE_NAME" \
  --region="$REGION" \
  --set-env-vars="GOOGLE_CLIENT_ID=${CLIENT_ID},GOOGLE_CLIENT_SECRET=${CLIENT_SECRET},OAUTH_REDIRECT_URL=${REDIRECT_URL}"

# --- Done ---
echo ""
echo "=== Deployment complete ==="
echo "  Service URL:  $SERVICE_URL"
echo "  Redirect URI: $REDIRECT_URL"
echo ""
echo "  IMPORTANT: Add this redirect URI to your OAuth credentials:"
echo "  ${REDIRECT_URL}"
echo ""
echo "  Go to: https://console.cloud.google.com/apis/credentials"
echo "  Edit your OAuth client → Add '${REDIRECT_URL}' as an authorized redirect URI"
