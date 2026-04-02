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
    eusc-de-east-1
  )


if [[ -z "${AWS_PROFILE}" ]]; then
    export AWS_PROFILE="personal"
fi

# Set AWS_PROFILE_EUSC for European Sovereign Cloud regions (can be overridden via environment)
if [[ -z "${AWS_PROFILE_EUSC}" ]]; then
    export AWS_PROFILE_EUSC="esc_personal"
fi

echo "Using AWS_PROFILE: ${AWS_PROFILE}"
echo "Using AWS_PROFILE_EUSC: ${AWS_PROFILE_EUSC}"

binary_name="sumologic-extension"

ARCHITECTURES=(
  amd64
  arm64
)
layer_version=1
for arch in "${ARCHITECTURES[@]}"; do

    layer_name="${binary_name}-${arch}"

    for region in "${AWS_REGIONS[@]}"; do
        # Auto-detect profile based on region prefix
        if [[ "${region}" =~ ^eusc- ]]; then
          profile="${AWS_PROFILE_EUSC}"
        else
          profile="${AWS_PROFILE}"
        fi

        echo "Deleting from region ${region} using profile ${profile}"

        # Dynamically get the partition for the region from AWS
        caller_arn=$(aws sts get-caller-identity --region ${region} --profile ${profile} --query 'Arn' --output text)
        partition=$(echo ${caller_arn} | cut -d':' -f2)

        echo "Layer Arn: arn:${partition}:lambda:${region}:<accountId>:layer:${layer_name}:${layer_version} deleted from Region ${region}"
        aws lambda delete-layer-version --layer-name ${layer_name} --version-number ${layer_version} --region ${region} --profile ${profile}
    done
done
