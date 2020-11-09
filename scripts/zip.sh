# Assuming the zip.sh script is run from inside the scripts folder
jqCmd=$(jq --version)
status=$?
if [ $status -eq 127 ]; then
	echo "Please install jq (Mac: brew install jq Or Ubuntu: apt-get install jq)"
  exit 1
fi

# clean up of old target directories
cd ..
TARGET_DIR=target
if [ -d "$TARGET_DIR" ]; then
  echo "removing old $TARGET_DIR"
  rm -r $TARGET_DIR;
fi

# Add GO packages to GOPATH. Not needed if you are using Go modules
# export GOPATH=${HOME}/GO:${PATH}:$(pwd)

echo "Creating an binary executable using the go build command for Linux Systems."
mkdir -p $TARGET_DIR/extensions
mkdir -p $TARGET_DIR/zip

env GOOS=linux go build -o $TARGET_DIR/extensions/sumologic-extension lambda-extensions/sumologic-extensions.go

status=$?
if [ $status -ne 0 ]; then
	echo "Binary Generation Failed"
  exit 1
fi
chmod +x $TARGET_DIR/extensions/sumologic-extension

echo "Creating the Zip file binary in extension folder."
cd $TARGET_DIR
zip -r zip/sumologic-extension.zip extensions/
status=$?
if [ $status -ne 0 ]; then
	echo "Zip Generation Failed"
  exit 1
fi
cd ..

echo "Create lambda Layer from the new ZIP file in the provided AWS_PROFILE aws account."
if [[ -z "${AWS_PROFILE}" ]]; then
  export AWS_PROFILE="personal"
fi
declare -a AWS_REGIONS=("us-east-1")

echo "Using AWS_PROFILE: ${AWS_PROFILE}"

# We have layer name as sumologic-extension. Please change name for local testing.
export layer_name="sumologic-extension"

for region in "${AWS_REGIONS[@]}"
do
    layer_arn=$(aws lambda publish-layer-version --layer-name ${layer_name} \
    --description "The SumoLogic Extension collects lambda logs and send it to Sumo Logic." \
    --license-info "MIT" --zip-file fileb://$(pwd)/$TARGET_DIR/zip/sumologic-extension.zip \
    --profile ${AWS_PROFILE} --region ${region} | jq -r .LayerVersionArn)
    echo "${layer_arn} Layer deployed to Region ${region}"

    # releasing prod version
    # aws lambda add-layer-version-permission --layer-name ${layer_name}  --statement-id ${layer_name}-prod --version-number $(echo -n $layer_arn | tail -c 1) --principal '*' --action lambda:GetLayerVersion --region ${region}

    # giving permission to content
    # aws lambda add-layer-version-permission --layer-name ${layer_name}  --statement-id ${layer_name}-dev --version-number $(echo -n $layer_arn | tail -c 1) --principal '956882708938' --action lambda:GetLayerVersion --region ${region}
done
