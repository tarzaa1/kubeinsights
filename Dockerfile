FROM golang:1.21.5 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /kubeinsights ./cmd/main

FROM ubuntu:latest

RUN apt-get update && \
    apt-get install -y ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /kubeinsights .

CMD ["./kubeinsights"]
