# Add GO packages to GOPATH
cd ..
export GOPATH=${HOME}/GO:${PATH}:$(pwd)

# Create an binary executable using the go build command for Linux Systems.
rm resources/extensions/sumologic-extension
env GOOS=linux go build -o resources/extensions/sumologic-extension src/sumologic-extensions.go

# Create the Zip file binary in extension folder.
rm resources/zip/sumologic-extension.zip
cd resources/
zip -r zip/sumologic-extension.zip extensions/ -x "*/.*"
cd ..

# Create lambda Layer from the new ZIP file in the provided AWS_PROFILE aws account.
export AWS_PROFILE="personal"
declare -a AWS_REGIONS=("us-east-1")
# "us-east-2" "us-west-1" "us-west-2" "ap-south-1" "ap-northeast-2" "ap-southeast-1" "ap-southeast-2" "ap-northeast-1" "ca-central-1" "eu-central-1" "eu-west-1" "eu-west-2" "eu-west-3" "eu-north-1s" "sa-east-1" "ap-east-1s" "af-south-1s" "eu-south-1" "me-south-1s")

# We have layer name as sumologic-extension.
export layer_name="sumologic-extension"

for region in "${AWS_REGIONS[@]}"
do
    aws lambda publish-layer-version --layer-name ${layer_name} \
    --description "The SumoLogic Extension collects lambda logs and send it to Sumo Logic." \
    --license-info "MIT" --zip-file fileb://$(pwd)/resources/zip/sumologic-extension.zip \
    --profile ${AWS_PROFILE} --region ${region}
    echo "Layer deployed to Regions ${region}"
done