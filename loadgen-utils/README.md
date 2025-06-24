# Load Generation Utilities (`loadgen-utils`)

This directory contains a suite of Go services designed to help configure and execute load generation tasks, primarily targeting web services.

## Services

### 1. `loadgenConfig`

*   **Purpose**: Provides a simple web interface to input parameters for a load generation task.
*   **Functionality**:
    *   Accepts user input for:
        *   Target URL (required)
        *   Target CPU utilization % (optional, default 0)
        *   QPS (Queries Per Second, optional, default 1)
        *   Duration in seconds (optional, default 1s)
    *   Stores these parameters as documents in a Google Cloud Firestore collection named `loadgen-configs`.
*   **Usage**: Run the service and access its web page (default port 8080) to submit new load generation configurations.

### 2. `requestLoadgen`

*   **Purpose**: Reads the configurations from Firestore and executes the load generation by sending HTTP GET requests.
*   **Functionality**:
    *   Reads all documents from the `loadgen-configs` Firestore collection.
    *   For each configuration:
        *   Sends HTTP GET requests to the specified `TargetURL`.
        *   If `TargetCPU` is provided, it appends it as a `targetCpuPct` query parameter.
        *   If `Duration` is provided, it appends it as a `durationS` query parameter (and also uses it to control the run length of the test).
        *   Sends requests asynchronously at the frequency defined by `QPS` for the given `Duration`.
    *   Logs information about the load generation process.
    *   Includes placeholder logic to eventually query Google Cloud Monitoring for the `run.googleapis.com/request_count` metric to compare configured QPS with actual QPS. (This feature requires further development to map target URLs to specific monitored Cloud Run services).

## Use Case

These utilities can be used together to:

1.  Easily define multiple load test scenarios via a web UI (`loadgenConfig`).
2.  Automatically execute these tests by running `requestLoadgen`, which will pick up all defined configurations.

This is useful for performance testing, validating auto-scaling behavior, or general system stress testing.

## Setup and Running

Please refer to `AGENTS.md` in this directory for detailed instructions on environment setup, building, and running these services.
```
