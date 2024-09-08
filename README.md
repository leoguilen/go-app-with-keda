# Go App with Kubernetes and KEDA

This project demonstrates how to build and deploy a Go application in a Kubernetes cluster and set up event-driven autoscaling using KEDA (Kubernetes Event-Driven Autoscaling).

## Prerequisites

Before getting started, you'll need:

- [VSCode](https://code.visualstudio.com/) installed with [Dev Containers](https://containers.dev/) extension
- [Docker](https://www.docker.com/get-started) installed

## Environment Setup

1. **Clone this repository:**

    ```bash
    git clone https://github.com/leoguilen/go-app-with-keda.git
    cd go-app-with-keda
    ```

2. **Open project in dev container:**

    In VSCode, open command tab and execute `Dev Container: Reopen in Container`.

3. **Setup local k8s cluster:**

    ```bash
    k3d cluster create --registry-create k3d-registry:0.0.0.0:5000
    
    echo '127.0.0.1 k3d-registry' | sudo tee -a /etc/hosts
    
    kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.15.1/keda-2.15.1.yaml
    ```

## Building the Application

1. **Build and Push the Docker images:**

    ```bash
    docker buildx build --build-arg="APP_NAME=consumer" -t k3d-registry:5000/consumer --push .
    
    docker buildx build --build-arg="APP_NAME=producer" -t k3d-registry:5000/producer --push .
    ```

## Deployment to Kubernetes

1. **Apply deployments for the applications:**

    ```bash
    kubectl apply -f deploy/k8s/ -R
    ```

2. **Verify the deployment:**

    ```bash
    kubectl get pod
    ```

## Testing Autoscaling

To test autoscaling, generate events that cause changes in the configured metric. Check logs and the number of replicas to ensure KEDA is scaling the application as expected.

You can use the helper script to generate orders:

```bash
./scripts/generate_orders.sh
```
