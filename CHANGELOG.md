# CHANGELOG
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html)


## [Unreleased]



## [1.0.0] - 2020-11-09

### Features
- Add chunking and compress all Lambda logs.
- Send all Lambda logs to Sumo Logic to an HTTPS Source endpoint using multiple consumers in goroutine.
- If the extension gets a throttling message from Sumo Logic (429, 503 or 504 HTTP response code), then the extension should write the log messages to an S3 bucket specified by the customer.

