package main

import (
	"log"
	"os"

	"github.com/tarzaa1/kubeinsights/pkg/publisher"

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

	client := publisher.CreateHederaClient(networkAddress, networkAccountId, operatorAccountId, operatorPrivateKey, mirrorAddress)

	newAccountId, newPrivateKey := publisher.NewHederaAccount(client)

	config := publisher.HederaConfig{}
	config.Network.Address = networkAddress
	config.Network.AccountId = networkAccountId
	config.Operator.AccountId = newAccountId.String()
	config.Operator.PrivateKey = newPrivateKey.String()
	config.MirrorNode = mirrorAddress

	publisher.WriteHederaConfig(config, "config.json")
}
