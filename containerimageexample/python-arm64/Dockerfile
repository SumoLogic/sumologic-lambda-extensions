# Define global args
ARG FUNCTION_DIR="app"
ARG RUNTIME_VERSION="3.13.7"
# Stage 1 - bundle base image + runtime
# Grab a fresh copy of the ARM 64 based image
FROM arm64v8/python:${RUNTIME_VERSION}-slim-bookworm AS python-slim-bookworm

# Stage 2 - build function and dependencies
FROM python-slim-bookworm AS build-image
# Install aws-lambda-cpp build dependencies
RUN apt-get update && \
  apt-get install -y \
  g++ \
  make \
  cmake \
  unzip \
  libcurl4-openssl-dev

# Include global args in this stage of the build
ARG FUNCTION_DIR
ARG RUNTIME_VERSION
# Create function directory
RUN mkdir -p ${FUNCTION_DIR}
# Copy handler function
COPY app/* ${FUNCTION_DIR}
# Optional – Install the function's dependencies
# RUN python${RUNTIME_VERSION} -m pip install -r requirements.txt --target ${FUNCTION_DIR}
# Install Lambda Runtime Interface Client for Python
RUN pip install awslambdaric --target ${FUNCTION_DIR}

# Stage 3 - final runtime image
# Grab a fresh copy of the Python image
FROM python-slim-bookworm
# Include global arg in this stage of the build
ARG FUNCTION_DIR
# Set working directory to function root directory it also creates the directory and cd to that directory
WORKDIR ${FUNCTION_DIR}
# Copy in the built dependencies
COPY --from=build-image ${FUNCTION_DIR} ./
# (Optional) Add Lambda Runtime Interface Emulator and use a script in the ENTRYPOINT for simpler local runs
ADD https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie /usr/bin/aws-lambda-rie
# Create the target directory for extensions
RUN mkdir -p /opt/extensions

# Extract the Sumo Logic Lambda extension package
ADD sumologic-extension-arm64.tar.gz /opt/extensions/

# Clean up any hidden files from the extracted content
RUN find /opt/extensions -type f -name ".*" -delete
COPY entry.sh /
RUN chmod 755 /usr/bin/aws-lambda-rie /entry.sh
ENTRYPOINT [ "/entry.sh" ]
CMD [ "app.handler" ]
