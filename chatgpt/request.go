package chatgpt

import (
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

type Request struct {
	chatMsg []*ChatMsg
	sync.RWMutex
}

func NewRequest() *Request {
	req := &Request{
		chatMsg: make([]*ChatMsg, 0),
	}
	return req
}

// return msg_id
func (req *Request) PutSystemMsg(content, name string) string {
	req.Lock()
	defer req.Unlock()

	msg := NewChatMsg(openai.ChatMessageRoleSystem, content, name)
	req.chatMsg = append(req.chatMsg, msg)
	return msg.id
}

// renturn msg_id, parent_id
func (req *Request) PutUserMsg(content, name string) (string, string) {
	req.Lock()
	defer req.Unlock()
	var parentId string
	if len(req.chatMsg) > 0 {
		parentId = req.chatMsg[len(req.chatMsg)-1].id
	}
	msg := NewChatMsg(openai.ChatMessageRoleUser, content, name)
	req.chatMsg = append(req.chatMsg, msg)
	return msg.id, parentId
}

func (req *Request) PopMsg() *ChatMsg {
	req.Lock()
	defer req.Unlock()

	msg := req.chatMsg[len(req.chatMsg)-1]
	req.chatMsg = req.chatMsg[:len(req.chatMsg)-1]
	return msg

}

func (req *Request) SetResStream(id string, text string, responseStream *openai.ChatCompletionStreamResponse) {
	req.Lock()
	defer req.Unlock()

	for _, v := range req.chatMsg {
		if v.id == id {
			v.resText = text
			v.responseStream = responseStream
			return
		}
	}
}

func (req *Request) SetRes(id string, response *openai.ChatCompletionResponse) {
	if response == nil {
		return
	}
	req.Lock()
	defer req.Unlock()

	for _, v := range req.chatMsg {
		if v.id == id {
			v.response = response
			return
		}
	}
}

// get message for send ask
func (req *Request) GetMessage(options ...string) []openai.ChatCompletionMessage {
	req.Lock()
	defer req.Unlock()
	var parentId string
	if len(options) > 0 {
		parentId = options[0]
	}
	messages := []openai.ChatCompletionMessage{}
	var done bool
	for _, v := range req.chatMsg {
		messages = append(messages, *v.request)
		if !done && v.resText != "" {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: v.resText,
			})
		}
		if parentId != "" && v.id == parentId {
			done = true
			continue
		}
		if done {
			break
		}
	}
	return messages
}
