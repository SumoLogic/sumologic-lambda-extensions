#!/bin/bash

LAYER_NAME="sumologic-extension-arm64"

AWS_REGIONS=(
    us-east-1
    us-east-2
    eu-north-1
    ap-south-1
    eu-west-3
    eu-west-2
    eu-south-1
    eu-west-1
    ap-northeast-2
    me-south-1
    ap-northeast-1
    sa-east-1
    ca-central-1
    ap-east-1
    ap-southeast-1
    ap-southeast-2
    eu-central-1
    us-west-1
    us-west-2
)

echo "Fetching latest version of layer: $LAYER_NAME"
echo "================================================="

for region in "${AWS_REGIONS[@]}"; do
    latest_version=$(aws lambda list-layer-versions \
        --layer-name "$LAYER_NAME" \
        --region "$region" \
        --query 'max_by(LayerVersions, &Version).Version' \
        --output text 2>/dev/null)

    if [ "$latest_version" != "None" ] && [ -n "$latest_version" ]; then
        echo "Region: $region -> Latest version: $latest_version"
    else
        echo "Region: $region -> Layer not found"
    fi
done
