package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	sub "kubeinsights/hedera"

	"github.com/hashgraph/hedera-sdk-go/v2"
)

func main() {

	//Create your local client
	// node := make(map[string]hedera.AccountID, 1)
	// node["10.18.1.35:50211"] = hedera.AccountID{Account: 3}

	// mirrorNode := []string{"10.18.1.35:5600"}

	// client := hedera.ClientForNetwork(node)
	// client.SetMirrorNetwork(mirrorNode)

	// accountId, err := hedera.AccountIDFromString("0.0.1002")
	// privateKey, err := hedera.PrivateKeyFromString("302e020100300506032b6570042204206dc33599b4a18682e0a3c058d2dd4de41888da8474ceae94c7983bfef1ea2661")
	// client.SetOperator(accountId, privateKey)

	client, err := sub.ClientFromFile("sub_config.json")

	// Get the topic ID from the transaction receipt
	topicID, err := hedera.TopicIDFromString("0.0.1003")

	if err != nil {
		println(err.Error(), ": error creating topic")
		return
	}

	//Log the topic ID to the console
	fmt.Printf("topicID: %v\n", topicID)

	// Subscribe to the topic
	_, err = hedera.NewTopicMessageQuery().
		SetTopicID(topicID).
		Subscribe(client.Client, func(message hedera.TopicMessage) {
			t2 := time.Now()
			var record map[string]interface{}
			err := json.Unmarshal(message.Contents, &record)
			if err != nil {
				log.Fatal(err)
			}
			timestamp := record["timestamp"].(string)
			t1, err := time.Parse(time.RFC3339Nano, timestamp)
			latency := t2.Sub(t1)
			// consenus_latency := message.ConsensusTimestamp.Sub(t1)
			// string(message.Contents)
			fmt.Println(message.ConsensusTimestamp.String(), "received topic message with latency ", latency.Milliseconds(), "\r")
		})

	if err != nil {
		println(err.Error(), ": error subscribing to topic")
		return
	}

	// Prevent the program from exiting to display the message from the mirror node to the console
	for {
		time.Sleep(30 * time.Second)
	}

}
