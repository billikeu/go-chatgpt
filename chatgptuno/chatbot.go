package chatgptuno

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/gjson"
)

type ChatGPTUnoBot struct {
	cfg            *ChatGPTUnoConfig
	jar            tls_client.CookieJar
	conversationId string
	parentId       string
	convMapping    *Mapping
}

func NewChatGPTUnoBot(cfg *ChatGPTUnoConfig) *ChatGPTUnoBot {
	chat := &ChatGPTUnoBot{
		cfg:            cfg,
		jar:            tls_client.NewCookieJar(),
		conversationId: "",
		parentId:       "",
		convMapping:    NewMapping(),
	}
	return chat
}

func (chat *ChatGPTUnoBot) BaseURL() string {
	// https://bypass.churchless.tech/api/
	// https://chat.openai.com/backend-api
	if chat.cfg.BaseUrl == "" {
		chat.cfg.BaseUrl = os.Getenv("CHATGPT_BASE_URL")
	}
	if chat.cfg.BaseUrl == "" {
		// endpoint = "https://chat.openai.com/backend-api/"
		chat.cfg.BaseUrl = "https://bypass.churchless.tech/api/"
	}
	return chat.cfg.BaseUrl
}

func (chat *ChatGPTUnoBot) SetAccessToken(accessToken string) {
	if accessToken == "" {
		return
	}
	headers := map[string]string{
		"Accept":                    "text/event-stream",
		"Authorization":             fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":              "application/json",
		"X-Openai-Assistant-App-Id": "",
		"Connection":                "close",
		"Accept-Language":           "en-US,en;q=0.9",
		"Referer":                   "https://chat.openai.com/chat",
	}
	log.Println(headers)

}

func (chat *ChatGPTUnoBot) defaultHeaders(accessToken string, keepAlive ...bool) http.Header {
	headers := http.Header{
		"Host":                      {"chat.openai.com"},
		"Origin":                    {"https://chat.openai.com/chat"},
		"Accept":                    {"text/event-stream"},
		"Authorization":             {fmt.Sprintf("Bearer %s", accessToken)},
		"Content-Type":              {"application/json"},
		"X-Openai-Assistant-App-Id": {""},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Referer":                   {"https://chat.openai.com/chat"},
		"User-Agent":                {UA},
	}
	if len(keepAlive) > 0 && keepAlive[0] {
		headers.Add("Connection", "keep-alive")
		headers.Add("Keep-Alive", "timeout=360, max=1000")
	} else {
		headers.Add("Connection", "close")
	}
	return headers
}

func (chat *ChatGPTUnoBot) Init() error {
	if chat.cfg.AccessToken == "" {
		return chat.Login()
	}
	return nil
}

func (chat *ChatGPTUnoBot) Login() error {
	if chat.cfg.AccessToken == "" && (chat.cfg.EmailAddr == "" || chat.cfg.Passwd == "") {
		return errors.New("auth info null")
	}

	auth := NewAuthenticator(&AuthConfig{
		EmailAddr: chat.cfg.EmailAddr,
		Passwd:    chat.cfg.Passwd,
		Proxy:     chat.cfg.Proxy,
	})
	defer chat.SetAccessToken(auth.AccessToken())

	if chat.cfg.SessionToken != "" {
		auth.SessionToken = chat.cfg.SessionToken
		err := auth.GetAccessToken()
		if err != nil {
			log.Println(err)
		}
		if err == nil && auth.AccessToken() != "" {
			chat.cfg.AccessToken = auth.AccessToken()
			return nil
		}
	}
	// login
	err := auth.Loin()
	if err != nil {
		return err
	}
	chat.cfg.AccessToken = auth.AccessToken()
	chat.cfg.SessionToken = auth.SessionToken
	return nil
}

// The standard ChatGPT model: text-davinci-002-render-sha Turbo (Default for free users)
func (chat *ChatGPTUnoBot) Ask(prompt, conversationId, parentId, model string, timeout int, callback func(chatRes *Response, err error)) error {
	var err error
	defer func() {
		if callback != nil && err != nil {
			callback(nil, err)
		}
	}()
	conversationId, parentId, err = chat.askBeforeInit(conversationId, parentId)
	if err != nil {
		return err
	}
	model = chat.getModelName(model)

	reqData := NewNextAction(prompt, conversationId, parentId, model)
	u, _ := url.Parse(chat.BaseURL())
	chat.jar.SetCookies(u, []*http.Cookie{
		{
			Name:  "library",
			Value: "revChatGPT",
		},
	})
	endpoint := fmt.Sprintf("%sconversation", chat.BaseURL())
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetBody(bytes.NewReader(reqData.Byte()))
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken, true))
	client.SetTimeout(timeout)
	resp, err := client.Post(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		err = fmt.Errorf("openai blocked your request %d", resp.StatusCode)
		return err
	}
	reader := bufio.NewReader(resp.Body)
	for {
		b, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("read ask data err:%s", err.Error())
			}
			break
		}
		if len(b) < 6 {
			continue
		}

		// log.Printf("[%s]", string(b))
		if string(b) == "data: [DONE]" {
			// log.Println("done .....")
			break
		}
		body := string(b)[6:]
		res := NewResponse(body)
		if res == nil {
			return fmt.Errorf("err response:%s", body)
		}
		if callback != nil {
			callback(res, nil)
		}
		if conversationId == "" {
			conversationId = res.ConversationID
		}
		convNode := chat.convMapping.GetConversationNode(conversationId)
		if convNode == nil {
			// new conversation
			chat.convMapping.SetConversationNode(conversationId, &ConversationNode{
				convId:   conversationId,
				parentId: res.Message.ID,
				title:    "",
			})
		} else {
			if res.Message.ID != "" {
				convNode.SetCurrentNode(res.Message.ID)
			}
		}
		if res.Message.EndTurn && res.Message.Author.Role == "assistant" {
			// change title: title == New chat or title == ""
			go chat.autoChangConversationTitle(conversationId, res.Message.ID)
			// log.Println("stop.............")
		}

		// log.Println(res.Message.Metadata.FinishDetails, "----------------------------------------------------", isPrefix)
	}
	// log.Println(conversationId, parentId)
	return nil
}

/*
	{
	    "items": [
	        {
	            "id": "bced487d-ed69-4203-9e0d-cb880ecd7a37",
	            "title": "Video Super-Resolution Website",
	            "create_time": "2023-03-28T07: 14: 37.333480",
	            "update_time": "2023-03-28T09: 45: 59"
	        }
	    ],
	    "total": 64,
	    "limit": 50,
	    "offset": 0,
	    "has_missing_conversations": false
	}
*/
func (chat *ChatGPTUnoBot) getConversations(offset int, limit int) error {
	endpoint := fmt.Sprintf("%sconversations?offset=%d&limit=%d", chat.BaseURL(), offset, limit)
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken))
	client.SetTimeout(60)
	// stream=True,
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(b)

	if resp.StatusCode != 200 {
		return fmt.Errorf("get conversations err:%s, %d", body, resp.StatusCode)
	}

	items := gjson.Parse(body).Get("items")
	for _, v := range items.Array() {
		// conversation
		conversationId := v.Get("id").String()
		if conversationId == "" {
			log.Println("need some update ?")
			continue
		}
		convNode := chat.convMapping.GetConversationNode(conversationId)
		if convNode == nil {
			chat.convMapping.SetConversationNode(conversationId, &ConversationNode{
				conversationInfo: v,
				convId:           conversationId,
			})
		} else {
			convNode.SetConversationInfo(v)
		}

	}
	return nil
}

func (chat *ChatGPTUnoBot) getMsgHistory(conversationId string) error {
	endpoint := fmt.Sprintf("%sconversation/%s", chat.BaseURL(), conversationId)
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken))
	client.SetTimeout(60)
	resp, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(b)
	if resp.StatusCode != 200 {
		return fmt.Errorf("get msg history err:%s, %d", body, resp.StatusCode)
	}
	convNode := chat.convMapping.GetConversationNode(conversationId)
	if convNode == nil {
		return errors.New("can not found conversation")
	}
	convNode.SetHistory(gjson.Parse(body))
	return nil
}

// Generate title for conversation
func (chat *ChatGPTUnoBot) genTitle(conversationId, messageId string) (title string, err error) {
	data := map[string]string{
		"message_id": messageId,
		"model":      "text-davinci-002-render",
	}
	b, err := json.Marshal(data)
	if err != nil {
		return title, fmt.Errorf("gen title err:%s", err.Error())
	}
	endpoint := fmt.Sprintf("%sconversation/gen_title/%s", chat.BaseURL(), conversationId)
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken))
	client.SetTimeout(60)
	client.SetBody(bytes.NewReader(b))
	resp, err := client.Post(endpoint)
	if err != nil {
		return title, fmt.Errorf("gen title err:%s", err.Error())
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return title, fmt.Errorf("gen title err:%s", err.Error())
	}
	body := string(b)
	if resp.StatusCode != 200 {
		return title, fmt.Errorf("gen title err:%s, %d", body, resp.StatusCode)
	}
	title = gjson.Parse(body).Get("title").String()
	// log.Println(title)
	return title, nil
}

// change title of conversation
func (chat *ChatGPTUnoBot) changeTitle(conversationId, title string) error {
	data := map[string]string{"title": title}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%sconversation/%s", chat.BaseURL(), conversationId)
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken))
	client.SetTimeout(60)
	client.SetBody(bytes.NewReader(b))
	resp, err := client.Patch(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(b)
	if resp.StatusCode != 200 {
		return fmt.Errorf("change title err:%s, %d", body, resp.StatusCode)
	}
	return nil
}

func (chat *ChatGPTUnoBot) DeleteConversation(conversationId string) error {
	endpoint := fmt.Sprintf("%sconversation/%s", chat.BaseURL(), conversationId)
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken))
	client.SetTimeout(60)
	client.SetBody(bytes.NewReader([]byte(`{"is_visible": false}`)))
	resp, err := client.Patch(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(b)
	if resp.StatusCode != 200 {
		return fmt.Errorf("delete conversation err:%s, %d", body, resp.StatusCode)
	}
	return nil
}

func (chat *ChatGPTUnoBot) ClearConversations() error {
	endpoint := fmt.Sprintf("%sconversations", chat.BaseURL())
	client := NewRequests(chat.jar)
	client.SetProxy(chat.cfg.Proxy)
	client.SetHeaders(chat.defaultHeaders(chat.cfg.AccessToken))
	client.SetTimeout(60)
	client.SetBody(bytes.NewReader([]byte(`{"is_visible": false}`)))
	resp, err := client.Patch(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body := string(b)
	if resp.StatusCode != 200 {
		return fmt.Errorf("clear conversation err:%s, %d", body, resp.StatusCode)
	}
	return nil
}

func (chat *ChatGPTUnoBot) autoChangConversationTitle(convId, msgId string) {
	if convId == "" || msgId == "" {
		return
	}
	convNode := chat.convMapping.GetConversationNode(convId)
	if convNode == nil {
		return
	}
	if convNode.Title() == "" {
		// refresh
		err := chat.getConversations(0, 50)
		if err != nil {
			log.Println(err)
			return
		}
	}
	if convNode.Title() != "New chat" {
		return
	}
	title, err := chat.genTitle(convId, msgId)
	if err != nil {
		log.Println(err)
		return
	}
	err = chat.changeTitle(convId, "gochat:"+title)
	if err != nil {
		log.Println(err)
		return
	}
}

func (chat *ChatGPTUnoBot) getModelName(model string) string {
	if model == "" {
		model = chat.cfg.Model
	}
	if model == "" {
		model = "text-davinci-002-render-sha"
	}
	return model
}

func (chat *ChatGPTUnoBot) askBeforeInit(conversationId, parentId string) (string, string, error) {
	// conversationId == ""
	if conversationId == "" {
		if parentId != "" {
			return conversationId, parentId, errors.New("conversation_id must be set once parent_id is set")
		}
		return conversationId, uuid.NewV4().String(), nil
	}
	// conversationId != ""
	if parentId == "" {
		// get conversation info from cache
		convNode := chat.convMapping.GetConversationNode(conversationId)
		if convNode != nil && convNode.CurrentNode() != "" {
			parentId = convNode.CurrentNode()
			return conversationId, parentId, nil
		}
		// get conversation info from network
		err := chat.getConversations(0, 50)
		if err != nil {
			return conversationId, parentId, err
		}
		convNode = chat.convMapping.GetConversationNode(conversationId)
		if convNode == nil {
			return conversationId, uuid.NewV4().String(), nil
		}
		// get msg history
		err = chat.getMsgHistory(conversationId)
		if err != nil {
			return conversationId, parentId, err
		}
		parentId = convNode.CurrentNode()
		return conversationId, parentId, nil

	}
	return conversationId, parentId, nil
}
