docker build -t lambda/hello-world-python:3.9-alpine3.12 .

## Command to run container
# docker run -p 9000:8080 lambda/hello-world-python:3.9-alpine3.12

## Command to test
# curl -XPOST "http://localhost:9000/2015-03-31/functions/function/invocations" -d '{}'

## Command to push image
ACCOUNT_ID=956882708938
aws ecr create-repository --repository-name hello-world-python-arm64 --image-scanning-configuration scanOnPush=true
docker tag lambda/hello-world-python:3.9-alpine3.12 "${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/"hello-world-python-arm64:latest
aws ecr get-login-password | docker login --username AWS --password-stdin "${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com"
docker push "${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/hello-world-python-arm64:latest"
