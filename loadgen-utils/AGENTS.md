# Agent Instructions for loadgen-utils

This document provides instructions for building, running, and maintaining the Go services within the `loadgen-utils` directory.

## Services

1.  **`loadgenConfig`**: A web service that provides a UI to input load generation parameters (Target URL, CPU, QPS, Duration) and stores them in Google Cloud Firestore.
2.  **`requestLoadgen`**: A service that reads configurations from Firestore and generates HTTP GET load against the specified target URLs. It aims to log configured QPS vs actual QPS (actual QPS from Cloud Monitoring is a future enhancement).

## Prerequisites

*   Go (version 1.18+ recommended)
*   Access to a Google Cloud Project.
*   `gcloud` CLI installed and authenticated (`gcloud auth application-default login` for local development).
*   Firestore API enabled in the GCP project.
*   Cloud Monitoring API enabled in the GCP project (for future `requestLoadgen` enhancements).

## Environment Variables

Both services require:

*   `GOOGLE_CLOUD_PROJECT`: Your Google Cloud Project ID.

`loadgenConfig` also uses:

*   `PORT`: The port on which the web service will listen (defaults to `8080`).

## Building the Services

Navigate to each service's directory and run `go build`:

```bash
# For loadgenConfig
cd loadgenConfig
go build .
cd ..

# For requestLoadgen
cd requestLoadgen
go build .
cd ..
```

## Running the Services

### `loadgenConfig`

1.  Ensure `GOOGLE_CLOUD_PROJECT` is set.
2.  Set `PORT` if you don't want to use `8080`.
3.  Run the executable:

    ```bash
    export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
    export PORT="8081" # Optional
    ./loadgenConfig/loadgenConfig
    ```
    Access the service at `http://localhost:PORT`.

### `requestLoadgen`

1.  Ensure `GOOGLE_CLOUD_PROJECT` is set.
2.  Run the executable:

    ```bash
    export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
    ./requestLoadgen/requestLoadgen
    ```
    This service runs, processes configurations, and then exits once all load generation tasks are complete.

## Firestore Data

*   Both services use a Firestore collection named `loadgen-configs`.
*   Ensure the service account or user credentials used have read/write permissions to Firestore in the specified project.

## Cloud Monitoring (for `requestLoadgen`)

*   The current implementation of `requestLoadgen` has placeholder logic for fetching the `run.googleapis.com/request_count` metric.
*   To fully implement this, the `ConfigParams` struct would need to include identifiers for the Cloud Run service (e.g., service name, region).
*   The service account/credentials need `monitoring.timeSeries.list` permission.

## Dependencies

Dependencies are managed by Go Modules. Run `go mod tidy` in each service's directory if you modify dependencies.

```bash
cd loadgenConfig
go mod tidy
cd ../requestLoadgen
go mod tidy
cd ..
```

## Code Conventions

*   Follow standard Go formatting (`gofmt`).
*   Include comments for public functions and complex logic.
*   Handle errors appropriately.
```
