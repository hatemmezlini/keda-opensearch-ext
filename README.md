# External Scaler for KEDA

## Introduction

A KEDA extenal scaler for Opensearch. Currently the ELasticsearch scaler in KEDA does not support Opensearch

## Prerequisites

Before you begin, ensure you have met the following requirements:

- **Kubernetes Cluster**: A running Kubernetes cluster.
- **KEDA**: Installed KEDA in your Kubernetes cluster.
- **Docker**: Docker installed on your local machine.
- **Go**: Go programming language installed (if you plan to modify the scaler code).

## How to Run Locally

To run the external scaler locally, follow these steps. For test purposes, you will have to set the environment variables loaded in main.go:27

1. **Clone the Repository**:
    ```bash
    git clone https://github.com/hatemmezlini/keda-opensearch-ext.git
    cd keda-opensearch-ext
    ```

2. **Install dependencies **:
    ```bash
    go mod tidy
    ```

3. **Test the Application locally**:
    ```bash
    go run main.go
    ```

## How to Build and Run Docker Image

To build and run the external scaler as a Docker container, follow these steps:

1. **Build the Docker Image**:
    ```bash
    docker build -t your-username/external-scaler-for-keda:latest .
    ```

2. **Run the Docker Container**:
    ```bash
    docker run -d -p 8080:8080 -p 6000:6000 --name keda-opensearch-ext -e ES_URL="http://localhost:9200" -e ES_USERNAME="keda" -e ES_PASSWORD="keda" opensearch-keda-externalscaler your-username/external-scaler-for-keda:latest
    ```

## Usage example
```
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: your-scaledobject-name
  namespace: your-namespace
spec:
  scaleTargetRef:
    name: your-scaledobject-name
    # kind: Statefulset (defaults to Deployment)
  pollingInterval: 30
  cooldownPeriod: 43200
  triggers:
    - type: external
      metadata:
        scalerAddress: opensearch-externalscaler.your-namespace.svc.cluster.local:6000
        unsafeSSL: "true"  # default is false
        index: "your-index-name-*"
        searchTemplateName: "search_template_name"
        targetValue: "5000"
        parameters: "param1=value1,param2=value2"

```

## Conclusion

Thank you for using the External Scaler for KEDA! This project aims to provide a flexible and powerful solution to extend KEDA's autoscaling capabilities. If you have any questions or encounter any issues, feel free to open an issue on the GitHub repository. Happy scaling!
