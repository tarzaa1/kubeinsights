package main

import (
	"fmt"
	"log"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	address := os.Getenv("KAFKA_BROKER_URL")
	groud_id := os.Getenv("KAFKA_GROUP_ID")
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": address,
		"group.id":          groud_id,
		"auto.offset.reset": "smallest"})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	topic := os.Getenv("KAFKA_TOPIC")
	topics := []string{topic}

	err = c.SubscribeTopics(topics, nil)
	if err != nil {
		fmt.Printf("Failed to subscribe to topics: %s\n", err)
		os.Exit(1)
	}
	run := true

	for run {
		ev := c.Poll(100)
		switch e := ev.(type) {
		case *kafka.Message:
			// application-specific processing
			fmt.Print(e)
		case kafka.Error:
			fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
			run = false
		default:
			// fmt.Printf("Ignored %v\n", e)
		}
	}
	c.Close()
}
