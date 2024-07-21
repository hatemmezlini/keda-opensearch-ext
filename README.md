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
    cd external-scaler-for-keda
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

## How to Install with Helm in Kubernetes

To deploy the external scaler using Helm, follow these steps:

1. **Add the Helm Chart Repository** (if you have a custom Helm repository):
    ```bash
    helm repo add my-repo https://charts.myrepo.com
    helm repo update
    ```

2. **Install the Helm Chart**:
    ```bash
    helm install external-scaler my-repo/external-scaler --set scalerSetting=value
    ```

   If you are using a local Helm chart, navigate to the chart directory and install:
    ```bash
    helm install external-scaler ./helm-chart --set scalerSetting=value
    ```

3. **Verify the Installation**:
    ```bash
    kubectl get pods -l app=external-scaler
    ```

   Ensure that the external scaler pod is running and healthy.

## Conclusion

Thank you for using the External Scaler for KEDA! This project aims to provide a flexible and powerful solution to extend KEDA's autoscaling capabilities. If you have any questions or encounter any issues, feel free to open an issue on the GitHub repository. Happy scaling!
