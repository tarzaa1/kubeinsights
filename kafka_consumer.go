package main

import (
	"fmt"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "10.18.1.35:29092,",
		"group.id":          "foo",
		"auto.offset.reset": "smallest"})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	topics := []string{"myTopic"}

	err = c.SubscribeTopics(topics, nil)
	run := true

	for run == true {
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
