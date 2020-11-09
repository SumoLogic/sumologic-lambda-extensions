# Contributing Guide

First of all, thanks for contributing!

## Reporting Issues

  - If you think you've found an issue with the extension, you can open a Github issue.

  - Feel free to reach out to us at sumodojo.slack.com.

## Development
* Fork the [repo](https://github.com/SumoLogic/sumologic-lambda-extensions) 🎉
* Create a feature branch ☕
* Run unit tests and confirm that it passes ⚡
* Commit your changes 📝
* Rebase your local changes against the master branch 💡
* Create new Pull Request.

## Building and Deploying a layer
    cd scripts/
    sh zip.sh

## Unit Testing

    go test sumoclient_test.go -v

## Integration Testing (Manual)

Add your layer to lambda by following [docs](https://help.sumologic.com/03Send-Data/Collect-from-Other-Data-Sources/Collect_Logs_from_AWS_Lambda_using_Lambda_Extension) and test the function manually. Confirm that logs are coming to Sumo Logic.
