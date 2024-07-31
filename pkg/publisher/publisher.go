package publisher

type Publisher interface {
	SubmitMessage(message []byte) string
}
