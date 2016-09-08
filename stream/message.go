package stream

type Message struct {
	Kind    int    `json:"kind"`
	Content string `json:"content"`
}
