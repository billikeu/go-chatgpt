package chatgpt

import (
	openai "github.com/sashabaranov/go-openai"
	uuid "github.com/satori/go.uuid"
)

type ChatMsg struct {
	id             string
	request        *openai.ChatCompletionMessage
	response       *openai.ChatCompletionResponse
	responseStream *openai.ChatCompletionStreamResponse
	resText        string
}

func NewChatMsg(role, content, name string) *ChatMsg {
	req := &ChatMsg{
		id: uuid.NewV4().String(),
		request: &openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
			Name:    name,
		},
		response:       &openai.ChatCompletionResponse{},
		responseStream: &openai.ChatCompletionStreamResponse{},
	}
	return req
}
