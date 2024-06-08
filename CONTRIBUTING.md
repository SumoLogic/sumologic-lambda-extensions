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

## Testing

   1> Unit Testing locally

    - Go to root folder and run "go test  ./..."

    - Go to lambda-extensions folder and run "go test  ./..."

   2> Testing with Lambda function

   Add the layer arn generated from build command output to your lambda function by following instructions in [docs](https://help.sumologic.com/03Send-Data/Collect-from-Other-Data-Sources/Collect_AWS_Lambda_Logs_using_an_Extension).Test by running the function manually. Confirm that logs are coming to Sumo Logic.

## Releasing the layers
  1. Change the *AWS_PROFILE* environment variable using below command. The profile should point to sumocontent aws account.
    `export AWS_PROFILE=<sumo content profile>`
  1. Update the layer version in *config/version.go*.
  1. Go to scripts folder
    `cd scripts/`
  1. Change the layer_name variable in zip.sh to avoid replacing the prod.
  1. Run below command
    `sh zip.sh`

### Github Release

  - The new extension binary and zip files gets released automatically after the tags are pushed using Github actions(Refer tagged-release in https://github.com/marvinpinto/action-automatic-releases).

     Run below commands to create and push tags

      git tag -a v<major.minor.patch> <commit_id>

      git push origin v<major.minor.patch>

  - Add the sumologic-extension-amd64.tar.gz and sumologic-extension-arm64.tar.gz files manually from the target folder generated after running zip.sh.
  - Update the release description with new layer arns and more details on what's changed.


### Upgrading to new golang versions
1. Make sure to install new go version. Preferably use [gvm](https://github.com/moovweb/gvm).
1. Update golang version in `go.mod` or run command `go mod edit -go <version ex 1.22>`.
1. Run `go mod tidy`. This will update `go.sum` file and clean up unwanted dependencies.
1. Install `golangci-lint` by running command `brew install golangci-lint`. Go to `lambda-extensions` directory and run `golangci-lint run`, this will check for deprecated methods. Check enabled linters using `golangci-lint linters` command.
1. Install `govulncheck`  by running command `go install golang.org/x/vuln/cmd/govulncheck@latest` and run `~/go/bin/govulncheck  -mode=binary  target/amd64/extensions/sumologic-extension`. this will find common security issues.
1. Run `go test  ./...` to run the unit tests
