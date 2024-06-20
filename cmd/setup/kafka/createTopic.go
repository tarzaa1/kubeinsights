package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	address := os.Getenv("KAFKA_BROKER_URL")
	topicID := os.Getenv("KAFKA_TOPIC")

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": address})
	if err != nil {
		log.Fatalf("Failed to create Admin client: %s", err)
	}

	topicSpecification := kafka.TopicSpecification{
		Topic:             topicID,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := adminClient.CreateTopics(ctx, []kafka.TopicSpecification{topicSpecification})
	if err != nil {
		log.Fatalf("Failed to create topic: %s", err)
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			log.Fatalf("Failed to create topic %s: %v", result.Topic, result.Error)
		} else {
			fmt.Printf("Successfully created topic %s\n", result.Topic)
		}
	}

	adminClient.Close()
}
