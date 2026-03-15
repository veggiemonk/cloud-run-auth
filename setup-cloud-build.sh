#!/usr/bin/env bash
set -euo pipefail

# One-time setup for Cloud Run source deploys (gcloud run deploy --source).
# Grants the default compute service account the permissions Cloud Build
# needs to read/write source archives in Cloud Storage.
#
# Run this once per project before the first --source deploy.
#
# Usage:
#   ./setup-cloud-build.sh

PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

echo "=== Setting up Cloud Build source deploy for project $PROJECT_ID ==="
echo "  Service account: $SA"
echo ""

# Enable required APIs
echo ">>> Enabling APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com

# Grant storage access so Cloud Build can read/write source archives
echo ">>> Granting storage.objectViewer to compute service account..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:${SA}" \
  --role=roles/storage.objectViewer \
  --condition=None \
  --quiet > /dev/null

echo ""
echo "=== Done. You can now run: gcloud run deploy <service> --source . ==="
