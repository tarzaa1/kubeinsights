package main

import (
	"fmt"

	"github.com/tarzaa1/kubeinsights/pkg/publisher"
)

func main() {

	config := publisher.ReadHederaConfig("config.json")

	hederaPublisher := publisher.NewHederaPublisher(config)

	topic, err := hederaPublisher.NewTopic("memo")
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("New topicID: %s\n", topic)
}
