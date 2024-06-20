package main

import (
	"fmt"
	"kubeinsights/pkg/hedera"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	networkAddress := os.Getenv("HEDERA_NODE_URL")
	networkAccountId := os.Getenv("NETWORK_ACCOUNT_ID")
	mirrorAddress := os.Getenv("HEDERA_MIRROR_NODE_URL")
	operatorAccountId := os.Getenv("OPERATOR_ACCOUNT_ID")
	operatorPrivateKey := os.Getenv("OPERATOR_PRIVATE_KEY")

	client, err := hedera.CreateClient(networkAddress, networkAccountId, operatorAccountId, operatorPrivateKey, mirrorAddress)

	if err != nil {
		panic(err.Error())
	}

	newAccountId, newPrivateKey, err := hedera.NewAccount(client)
	if err != nil {
		fmt.Println(err)
		return
	}

	config := hedera.Config{}
	config.Network.Address = networkAddress
	config.Network.AccountId = networkAccountId
	config.Operator.AccountId = newAccountId.String()
	config.Operator.PrivateKey = newPrivateKey.String()
	config.MirrorNode = mirrorAddress

	hedera.WriteConfig(config, "config.json")

	client, err = hedera.ClientFromFile("config.json")

	if err != nil {
		panic(err.Error())
	}

	topicID, err := hedera.NewTopic(client)
	if err != nil {
		return
	}

	fmt.Printf("New topicID: %v\n", topicID)
}
