# sumologic-lambda-extensions 

[![build-and-test][github-build-badge]][github-build]
[![GitHub release][github-release-badge]][github-release]
  
AWS Lambda Extensions lets you integrate Lambda with your favorite tools for monitoring, observability, security, and governance. Extensions enable you and your preferred tooling vendors to plug into Lambda’s lifecycle and integrate more deeply into the Lambda execution environment.

This repository contains SumoLogic AWS Lambda extension.

# AWS Layer Version

The Sumo Logic lambda extension is available as an AWS public Layer. The latest layer is:

For x86_64 use:

    arn:aws:lambda:<AWS_REGION>:956882708938:layer:sumologic-extension-amd64:<latest version from github release>

For arm64 use:

    arn:aws:lambda:<AWS_REGION>:956882708938:layer:sumologic-extension-arm64:<latest version from github release>


- AWS_REGION - Replace with your AWS Lambda Region.

### Receive logs during AWS Lambda execution time
All the logs that are not sent to Sumo Logic during the Execution phase of the AWS Lambda, are sent during the shutdown phase instead. For more details on phases on the lifecycle and AWS Lambda phases please see the[ AWS documentation ](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-context.html).

If you would like to always send logs during the execution phase however, you can add extra execution time via a sleep function at the end of lambda code, which will give your extension time to run and send logs to Sumo Logic. We recommend setting this to two seconds.

# Using Lambda extension in custom container images

Follow the instruction in [docs](https://help.sumologic.com/03Send-Data/Collect-from-Other-Data-Sources/Collect_AWS_Lambda_Logs_using_an_Extension#For_AWS_Lambda_Functions_Created_Using_Container_Images:)

Refer [containerimageexample](containerimageexample/python-arm64/) folder To see sample [Dockerfile](containerimageexample/python-arm64/Dockerfile) for python arm64 image.

# Contributing

  - To improve the existing app or reporting issues, follow instructions in [CONTRIBUTING](CONTRIBUTING.md)


# Community

   * You can also join our slack community at sumodojo.slack.com.

   * Here's the [CODE_OF_CONDUCT](CODE_OF_CONDUCT.md) guidelines to follow.

# Documentation

   * To know more about how to use this extension follow docs [here](https://help.sumologic.com/03Send-Data/Collect-from-Other-Data-Sources/Collect_Logs_from_AWS_Lambda_using_Lambda_Extension).
   * [AWS Lambda Extensions API](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html)

## Change Log

For Full Change Log, please visit [Releases](https://github.com/SumoLogic/sumologic-lambda-extensions/releases) page.

[github-build-badge]: https://github.com/SumoLogic/sumologic-lambda-extensions/workflows/build-and-test/badge.svg?branch=main

[github-build]: https://github.com/SumoLogic/sumologic-lambda-extensions/actions?query=workflow%3Abuild-and-test

[github-release-badge]: https://img.shields.io/github/release/sumologic/sumologic-lambda-extensions/all.svg?label=release

[github-release]: https://github.com/sumologic/sumologic-lambda-extensions/releases/latest


## Compiling
   
   `env GOOS=darwin go build -o "sumologic-extension" "lambda-extensions/sumologic-extension.go"`


## Building
   This script assumes you have aws cli already configured.

   - Go to scripts folder
   - Export Profile export AWS_PROFILE=<sumo content profile>
   - Change the layer_name variable in zip.sh to avoid replacing the prod.
   - Run below command
     `sh zip.sh`

## Release
Releasing new layer versions

- Go to scripts folder
- Export Profile export AWS_PROFILE=<sumo content profile>. The profile should point to sumocontnet aws account.
- Run below command
     `sh zip.sh`


- The new wheel package gets released automatically after the tags are pushed using Github actions(Refer tagged-release in https://github.com/marvinpinto/action-automatic-releases).

   Run below commands to create and push tags

    git tag -a v<major.minor.patch> <commit_id>

    git push origin v<major.minor.patch>

- Add the source files and binaries manually from the target folder generated after running zip.sh

## Testing

   1> Unit Testing locally
      
    - Go to root folder and run "go test  ./..."

    - Go to lambda-extensions folder and run "go test  ./..."

   2> Testing with Lambda function

   Add the layer arn generated from build command output to your lambda function by following instructions in [docs](https://help.sumologic.com/03Send-Data/Collect-from-Other-Data-Sources/Collect_AWS_Lambda_Logs_using_an_Extension).

