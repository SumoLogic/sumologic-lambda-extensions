# sumologic-lambda-extensions 

[![build-and-test][github-build-badge]][github-build]
[![GitHub release][github-release-badge]][github-release]
  
AWS Lambda Extensions lets you integrate Lambda with your favorite tools for monitoring, observability, security, and governance. Extensions enable you and your preferred tooling vendors to plug into Lambda’s lifecycle and integrate more deeply into the Lambda execution environment.

This repository contains SumoLogic AWS Lambda extension.

# AWS Layer Version

The Sumo Logic lambda extension is available as an AWS public Layer. The latest layer is:

    arn:aws:lambda:<AWS_REGION>:956882708938:layer:sumologic-extension:1

- AWS_REGION - Replace with your AWS Lambda Region.

### Receive logs during AWS Lambda execution time  
All the logs which are not sent to Sumo Logic during the execution of the AWS lambda, are sent to Sumo Logic during the [ShutDown](https://docs.aws.amazon.com/lambda/latest/dg/runtimes-context.html) of the AWS Lambda.

If you would like to send the Logs during the execution of the AWS lambda, you can add some extra execution time (using sleep at the end of lambda), which will give extension time to run and send the logs to Sumo Logic. We recommend adding a sleep time of around approx 1 - 2 seconds.


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
