#!/bin/bash -x
# Assuming the zip.sh script is run from inside the scripts folder

# clean up of old target directories
cd ..
TARGET_DIR=target
if [ -d "$TARGET_DIR" ]; then
  echo "removing old ${TARGET_DIR}"
  rm -r ${TARGET_DIR};
fi

# Add GO packages to GOPATH. Not needed if you are using Go modules
# export GOPATH=${HOME}/GO:${PATH}:$(pwd)

echo "Creating an binary executable using the go build command for Linux Systems."
binary_name="sumologic-extension"


ARCHITECTURES=(
  amd64
  arm64
)

for arch in "${ARCHITECTURES[@]}"; do

  echo "Creating an binary executable for $arch"
  extension_bin_dir="${TARGET_DIR}/${arch}/extensions"
  extension_zip_dir="${TARGET_DIR}/${arch}/zip"
  mkdir -p "${extension_bin_dir}"
  mkdir -p "${extension_zip_dir}"

  env GOOS="linux" GOARCH="$arch" go build -o "${extension_bin_dir}/${binary_name}" "lambda-extensions/${binary_name}.go"

  status=$?
  if [ $status -ne 0 ]; then
  	echo "Binary Generation Failed"
    exit 1
  fi
  chmod +x "${extension_bin_dir}/${binary_name}"

  echo "Creating the Zip file binary in extension folder."
  cd "${TARGET_DIR}/${arch}"
  zip -r "zip/${binary_name}.zip" "extensions/${binary_name}"
  tar -czvf "zip/${binary_name}-${arch}.tar.gz" -C extensions "${binary_name}"
  status=$?
  if [ $status -ne 0 ]; then
  	echo "Zip Generation Failed"
    exit 1
  fi
  cd -

  echo "Create lambda Layer from the new ZIP file in the provided AWS_PROFILE aws account."
  # Set AWS_PROFILE_EUSC for European Sovereign Cloud regions (can be overridden via environment)
  if [[ -z "${AWS_PROFILE_EUSC}" ]]; then
    export AWS_PROFILE_EUSC="esc_personal"
  fi

  AWS_REGIONS=(
    eusc-de-east-1
  )


  echo "Using AWS_PROFILE_EUSC: ${AWS_PROFILE_EUSC}"

  # We have layer name as sumologic-extension. Please change name for local testing.
  layer_name="${binary_name}-${arch}"

  for region in "${AWS_REGIONS[@]}"; do

      echo "Deploying to region ${region} using profile ${AWS_PROFILE_EUSC}"

      layer_version=$(aws lambda publish-layer-version --layer-name ${layer_name} \
      --description "The SumoLogic Extension collects lambda logs and send it to Sumo Logic." \
      --license-info "Apache-2.0" --zip-file fileb://$(pwd)/${extension_zip_dir}/${binary_name}.zip \
      --profile ${AWS_PROFILE_EUSC} --region ${region} --output text --query Version )

      # Dynamically get the partition for the region from AWS
      caller_arn=$(aws sts get-caller-identity --region ${region} --profile ${AWS_PROFILE_EUSC} --query 'Arn' --output text)
      partition=$(echo ${caller_arn} | cut -d':' -f2)

      echo "Layer Arn: arn:${partition}:lambda:${region}:<accountId>:layer:${layer_name}:${layer_version} deployed to Region ${region}"

      echo "Setting public permissions for layer version: ${layer_version}"
      aws lambda add-layer-version-permission --layer-name ${layer_name}  --statement-id ${layer_name}-prod --version-number $layer_version --principal '*' --action lambda:GetLayerVersion --region ${region} --profile ${AWS_PROFILE_EUSC}
      # aws lambda add-layer-version-permission --layer-name ${layer_name}  --statement-id ${layer_name}-dev --version-number ${layer_version} --principal '956882708938' --action lambda:GetLayerVersion --region ${region}
  done

done