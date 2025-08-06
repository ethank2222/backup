#!/bin/bash
# EXACT COPY from original workflow setup section

curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
sudo apt-get update && sudo apt-get install -y jq

# Ensure container exists
az storage container create \
  --account-name "$AZURE_STORAGE_ACCOUNT" \
  --account-key "$AZURE_STORAGE_KEY" \
  --name "$CONTAINER_NAME" \
  --public-access off || true 