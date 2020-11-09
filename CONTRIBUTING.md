# Contributing Guide

First of all, thanks for contributing!. Before contributing please read the [CODE_OF_CONDUCT](CODE_OF_CONDUCT.md) and [search the issue tracker](issues); your issue may have already been discussed

## Reporting Issues

  - If you think you've found an issue with the extension, you can open a Github issue.

  - Feel free to reach out to us at sumodojo.slack.com.

## Development
* Fork the [repo](https://github.com/SumoLogic/sumologic-lambda-extensions) 🎉
* Create a feature branch. ☕
* Run unit tests and confirm that it passes. ⚡
* Commit your changes. 📝
* Rebase your local changes against the master branch. 💡
* Create new Pull Request.

## Building
* To install build related dependencies use below command

  `env GO111MODULE=off go install <package>`.
* Always use `go mod tidy` to clean up unwanted dependencies.
* To generate the binary use below command

  ```go build -o target/extensions/sumologic-extension lambda-extensions/sumologic-extension.go```

## Unit Testing

    go test sumoclient_test.go -v

## Deploying the layer
  * Change the *AWS_PROFILE* environment variable.
  * Update the layer version in *config/version.go*.
  * Use below command for creating and deploying layer
  
        cd scripts/
        sh zip.sh


## Integration Testing (Manual)

Add your layer to lambda by following [docs](https://help.sumologic.com/03Send-Data/Collect-from-Other-Data-Sources/Collect_Logs_from_AWS_Lambda_using_Lambda_Extension) and test the function manually. Confirm that logs are coming to Sumo Logic.
