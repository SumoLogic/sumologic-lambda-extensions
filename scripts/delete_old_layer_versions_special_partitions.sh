AWS_REGIONS=(
    eusc-de-east-1
  )

# Set AWS_PROFILE_EUSC for European Sovereign Cloud regions (can be overridden via environment)
if [[ -z "${AWS_PROFILE_EUSC}" ]]; then
    export AWS_PROFILE_EUSC="esc_personal"
fi

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

        echo "Deleting from region ${region} using profile ${AWS_PROFILE_EUSC}"

        # Dynamically get the partition for the region from AWS
        caller_arn=$(aws sts get-caller-identity --region ${region} --profile ${AWS_PROFILE_EUSC} --query 'Arn' --output text)
        partition=$(echo ${caller_arn} | cut -d':' -f2)

        echo "Layer Arn: arn:${partition}:lambda:${region}:<accountId>:layer:${layer_name}:${layer_version} deleted from Region ${region}"
        aws lambda delete-layer-version --layer-name ${layer_name} --version-number ${layer_version} --region ${region} --profile ${AWS_PROFILE_EUSC}
    done
done