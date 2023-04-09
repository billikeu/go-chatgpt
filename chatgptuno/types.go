package chatgptuno

import (
	"encoding/json"
	"sync"

	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/gjson"
)

// requests msg begin +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type NextAction struct {
	Action          string          `json:"action"`
	Messages        []PromptMessage `json:"messages"`
	ConversationID  *string         `json:"conversation_id,omitempty"`
	ParentMessageID string          `json:"parent_message_id"`
	Model           interface{}     `json:"model"`
}

type PromptMessage struct {
	ID      string            `json:"id"`
	Role    string            `json:"role"`
	Author  map[string]string `json:"author"`
	Content ReqContent        `json:"content"`
}

type ReqContent struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

func NewNextAction(prompt string, conversationId, parentId string, model string) *NextAction {
	var pConversationId *string
	if conversationId != "" {
		pConversationId = &conversationId
	}
	nextAction := &NextAction{
		Action: "next",
		Messages: []PromptMessage{
			{
				ID:   uuid.NewV4().String(),
				Role: "user",
				Author: map[string]string{
					"role": "user",
				},
				Content: ReqContent{
					ContentType: "text",
					Parts:       []string{prompt},
				},
			},
		},
		ConversationID:  pConversationId,
		ParentMessageID: parentId,
		Model:           model,
	}
	return nextAction
}

func (nextAction *NextAction) String() string {
	b, err := json.Marshal(nextAction)
	if err != nil {
		return ""
	}
	return string(b)
}

func (nextAction *NextAction) Byte() []byte {
	b, err := json.Marshal(nextAction)
	if err != nil {
		return []byte(``)
	}
	return b
}

// requests msg end ---------------------------------------------------------------

// response msg begin +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

type Response struct {
	Message        Message `json:"message"`
	ConversationID string  `json:"conversation_id"`
	Error          *string `json:"error"`
	Raw            string
}

type Message struct {
	ID         string     `json:"id"`
	Author     Author     `json:"author"`
	CreateTime float64    `json:"create_time"`
	UpdateTime *float64   `json:"update_time,omitempty"`
	Content    ResContent `json:"content"`
	EndTurn    bool       `json:"end_turn"`
	Weight     float64    `json:"weight"`
	Metadata   Metadata   `json:"metadata"`
	Recipient  string     `json:"recipient"`
}

type Author struct {
	Role string      `json:"role"`
	Name interface{} `json:"name,omitempty"`
	// Metadata Metadata    `json:"metadata"`
}

type ResContent struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

type Metadata struct {
	MessageType   string       `json:"message_type"`
	ModelSlug     string       `json:"model_slug"`
	FinishDetails FinishDetail `json:"finish_details"`
}

type FinishDetail struct {
	Type string `json:"type"`
	Stop string `json:"stop"`
}

func NewResponse(text string) *Response {
	res := &Response{}
	err := json.Unmarshal([]byte(text), res)
	if err != nil {
		return nil
	}
	res.Raw = text
	return res
}

// response msg end -----------------------------------------------------------

// conversation node begin ++++++++++++++++++++++++++++++++++++++++++++++++++++++

type ConversationNode struct {
	title            string
	convId           string
	parentId         string
	conversationInfo gjson.Result
	historyInfo      gjson.Result
	sync.Mutex
}

func (node *ConversationNode) CurrentNode() string {
	node.Lock()
	defer node.Unlock()

	if node.parentId != "" {
		return node.parentId
	}
	node.parentId = node.historyInfo.Get("current_node").String()
	return node.parentId
}

func (node *ConversationNode) SetCurrentNode(parentId string) {
	node.Lock()
	defer node.Unlock()

	if parentId == "" {
		node.historyInfo = gjson.Result{}
	}
	node.parentId = parentId
}

func (node *ConversationNode) ConversationId() string {
	node.Lock()
	defer node.Unlock()

	if node.convId != "" {
		return node.convId
	}
	node.convId = node.conversationInfo.Get("id").String()
	return node.convId
}

func (node *ConversationNode) SetConversationId(conversationId string) {
	node.Lock()
	defer node.Unlock()

	if conversationId == "" {
		node.conversationInfo = gjson.Result{}
	}
	node.convId = conversationId
}

func (node *ConversationNode) SetHistory(history gjson.Result) {
	node.Lock()
	defer node.Unlock()

	node.historyInfo = history
}

func (node *ConversationNode) SetConversationInfo(info gjson.Result) {
	node.Lock()
	defer node.Unlock()

	node.conversationInfo = info
}

func (node *ConversationNode) Title() string {
	node.Lock()
	defer node.Unlock()

	if node.title != "" {
		return node.title
	}
	return node.conversationInfo.Get("title").String()
}

func (node *ConversationNode) SetTitle(title string) {
	node.Lock()
	defer node.Unlock()

	node.title = title
}

// conversation node end -------------------------------------------------------------------

// mapping begin ++++++++++++++++++++++++++++++++++++++++++++++++++

type Mapping struct {
	conversationMapping map[string]*ConversationNode
	sync.Mutex
}

func NewMapping() *Mapping {
	mapping := &Mapping{
		conversationMapping: make(map[string]*ConversationNode, 0),
	}
	return mapping
}

func (mapping *Mapping) GetConversationNode(convId string) *ConversationNode {
	mapping.Lock()
	defer mapping.Unlock()

	info := mapping.conversationMapping[convId]
	return info
}

func (mapping *Mapping) SetConversationNode(convId string, node *ConversationNode) {
	mapping.Lock()
	defer mapping.Unlock()

	mapping.conversationMapping[convId] = node
}

// mapping end ----------------------------------------------------------
