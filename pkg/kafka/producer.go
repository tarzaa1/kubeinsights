package kafka

import (
	"fmt"
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Client struct {
	*kafka.Producer
}

func Producer(address string) Client {

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": address})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	return Client{p}
}

func SubmitMessage(producer Client, topicID string, message []byte) string {

	delivery_chan := make(chan kafka.Event, 10000)
	err := producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topicID, Partition: kafka.PartitionAny},
		Value:          message},
		delivery_chan,
	)

	if err != nil {
		fmt.Printf("Produce failed: %s\n", err)
		return ""
	}

	e := <-delivery_chan
	m := e.(*kafka.Message)

	var status string

	if m.TopicPartition.Error != nil {
		status = fmt.Sprintf("Delivery failed: %v\n", m.TopicPartition.Error)
	} else {
		status = fmt.Sprintf("Delivered message to topic %s [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}
	close(delivery_chan)
	return status
}

func Ack(producer *kafka.Producer) {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				fmt.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
			} else {
				fmt.Printf("Successfully produced record to topic %s partition [%d] @ offset %v\n",
					*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
			}
		}
	}
}
