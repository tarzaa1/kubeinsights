package publisher

type Publisher interface {
	SubmitMessage(message []byte, topic string) string
}
