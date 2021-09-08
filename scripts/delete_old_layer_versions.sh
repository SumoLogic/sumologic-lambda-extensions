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


if [[ -z "${AWS_PROFILE}" ]]; then
    export AWS_PROFILE="personal"
fi
echo "Using AWS_PROFILE: ${AWS_PROFILE}"

binary_name="sumologic-extension"

ARCHITECTURES=(
  amd64
  arm64
)
layer_version=1
for arch in "${ARCHITECTURES[@]}"; do

    layer_name="${binary_name}-${arch}"

    for region in "${AWS_REGIONS[@]}"; do
        echo "Layer Arn: arn:aws:lambda:${region}:<accountId>:layer:${layer_name}:${layer_version} deleted from Region ${region}"
        aws lambda delete-layer-version --layer-name ${layer_name} --version-number ${layer_version} --region ${region}
    done
done
