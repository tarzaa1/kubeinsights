# kubeinsights

`kubeinsights` is a Go project that runs on kubernetes cluster control plane node, it fetches the state of the cluster via the API Server, forwards the state to a Kafka or a Hedera DLT topic, then watches live events from the cluster (e.g., add/delete pod) and relays them to the same topic.

## Prerequisites
- Go v1.21
- Docker

## Installation

1. Clone the repository

## Setup

1. Create a `.env` file from the provided `example.env`:
    ```sh
    cp example.env .env
    ```

2. Edit the `.env` file to configure the necessary enviornmnet variables for Kafka or Hedera.

## Usage

1. Run kubeinsights dev container, kafka, and zookeeper:
    ```sh
    docker compose up -d
    ```

2. Attach to running kubeinsights container via docker exec (Alternatively use vscode Dev Containers extension):
    ```sh
    docker exec -it -w /kubeinsights kubeinsights-kubeinsights-1 /bin/bash
    ```

3. Create Kafka topic:
    ```sh
    go run cmd/setup/kafka/createTopic.go
    ```

4. Run application:
    ```sh
    go run cmd/main/main.go
    ```




