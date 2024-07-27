package publisher

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashgraph/hedera-sdk-go/v2"
)

type HederaPublisher struct {
	Client *hedera.Client
	Topics map[string]*hedera.TopicID
}

func (p HederaPublisher) SubmitMessage(topicID string, message []byte) string {

	if topic, exits := p.Topics[topicID]; exits {
		submitMessage, err := hedera.NewTopicMessageSubmitTransaction().
			SetMessage(message).
			SetTopicID(*topic).
			SetMaxChunks(40).
			Execute(p.Client)
		if err != nil {
			panic(err.Error())
		}

		receipt, err := submitMessage.GetReceipt(p.Client)

		if err != nil {
			println(err.Error())
			return ""
		}

		transactionStatus := receipt.Status
		return transactionStatus.String()
	} else {
		return fmt.Sprintf("The topic %s does not exist", topicID)
	}
}

func (p *HederaPublisher) NewTopic(topicID string) {

	transactionResponse, err := hedera.NewTopicCreateTransaction().
		SetTopicMemo(topicID).
		Execute(p.Client)
	if err != nil {
		panic(err.Error())
	}

	transactionReceipt, err := transactionResponse.GetReceipt(p.Client)
	if err != nil {
		panic(err.Error())
	}

	p.Topics[topicID] = transactionReceipt.TopicID
}

func (p HederaPublisher) NewAccount() (hedera.AccountID, hedera.PrivateKey) {

	privateKey, err := hedera.PrivateKeyGenerateEd25519()
	if err != nil {
		panic(err.Error())
	}

	publicKey := privateKey.PublicKey()

	fmt.Printf("Private key: %v\n", privateKey.String())
	fmt.Printf("Public key: %v\n", publicKey.String())

	newAccount, err := hedera.NewAccountCreateTransaction().
		SetKey(publicKey).
		SetInitialBalance(hedera.NewHbar(1000)).
		Execute(p.Client)

	if err != nil {
		panic(err.Error())
	}

	receipt, err := newAccount.GetReceipt(p.Client)
	if err != nil {
		panic(err.Error())
	}

	newAccountId := receipt.AccountID
	fmt.Printf("accountID: %v\n", newAccountId)

	return *newAccountId, privateKey
}

type Config struct {
	Network struct {
		AccountId string `json:"accountId"`
		Address   string `json:"address"`
	} `json:"network"`
	Operator struct {
		AccountId  string `json:"accountId"`
		PrivateKey string `json:"privateKey"`
	} `json:"operator"`
	MirrorNode string `json:"mirrorNetwork"`
}

func ReadConfig(filepath string) Config {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		panic(err.Error())
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err.Error())
	}

	var config Config

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		panic(err.Error())
	}

	return config
}

func WriteConfig(config Config, filePath string) {

	jsonData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		panic(err.Error())
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		panic(err.Error())
	}
}

func CreateClient(networkAddress string, networkAccountId string, operatorAccountId string, operatorPrivateKey string, mirrorAddress string) *hedera.Client {

	node := make(map[string]hedera.AccountID, 1)
	networkAccountID, err := hedera.AccountIDFromString(networkAccountId)
	if err != nil {
		panic(err.Error())
	}

	node[networkAddress] = networkAccountID

	mirrorNode := []string{mirrorAddress}

	client := hedera.ClientForNetwork(node)
	client.SetMirrorNetwork(mirrorNode)

	accountId, err := hedera.AccountIDFromString(operatorAccountId)
	if err != nil {
		panic(err.Error())
	}
	privateKey, err := hedera.PrivateKeyFromString(operatorPrivateKey)
	if err != nil {
		panic(err.Error())
	}

	client.SetOperator(accountId, privateKey)

	return client
}

func ClientFromFile(filePath string) *hedera.Client {
	config := ReadConfig(filePath)

	networkAddress := config.Network.Address
	networkAccountId := config.Network.AccountId
	mirrorAddress := config.MirrorNode
	operatorAccountId := config.Operator.AccountId
	operatorPrivateKey := config.Operator.PrivateKey

	client := CreateClient(networkAddress, networkAccountId, operatorAccountId, operatorPrivateKey, mirrorAddress)

	return client
}
