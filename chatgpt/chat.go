package chatgpt

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/billikeu/go-chatgpt/common"
	"github.com/billikeu/go-chatgpt/params"

	openai "github.com/sashabaranov/go-openai"
)

type ChatGPTConversion struct {
	secretKey string //openai api
	botConfig openai.ClientConfig
	client    *openai.Client
	requst    *Request
	proxy     string
}

func NewChatGPTConversion(secretKey string) *ChatGPTConversion {
	chat := &ChatGPTConversion{
		secretKey: secretKey,
		botConfig: openai.DefaultConfig(secretKey),
		requst:    NewRequest(),
	}
	return chat
}

/*
set proxy
chat.SetProxy("socks5://127.0.0.1:3126")
chat.SetProxy("http://127.0.0.1:3127")
*/
func (chat *ChatGPTConversion) SetProxy(proxy string) error {
	if err := chat.setSocksProxy(proxy); err != nil {
		return err
	}
	if err := chat.setHttpProxy(proxy); err != nil {
		return err
	}
	chat.proxy = proxy
	return nil
}

func (chat *ChatGPTConversion) setHttpProxy(proxy string) error {
	if !strings.HasPrefix(proxy, "http") {
		return nil
	}
	transport := &http.Transport{}
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return err
	}
	transport.Proxy = http.ProxyURL(proxyUrl)
	chat.botConfig.HTTPClient = &http.Client{
		Transport: transport,
	}
	return nil
}

// set socks proxy
func (chat *ChatGPTConversion) setSocksProxy(proxy string) error {
	if !strings.HasPrefix(proxy, "socks5") {
		return nil
	}
	transport := &http.Transport{}
	dialContext, err := common.NewDialContext(proxy[9:])
	if err != nil {
		panic(err)
	}
	transport.DialContext = dialContext
	chat.botConfig.HTTPClient = &http.Client{
		Transport: transport,
	}
	return nil
}

// set base URL, default openai URL
func (chat *ChatGPTConversion) SetBaseURL(baseURL string) {
	if baseURL != "" {
		chat.botConfig.BaseURL = baseURL
	}
}

// init client
func (chat *ChatGPTConversion) Init() error {
	chat.client = openai.NewClientWithConfig(chat.botConfig)
	return nil
}

// set system role message
func (chat *ChatGPTConversion) SetSystemMsg(content string) {
	chat.requst.PutSystemMsg(content, "")
}

/*
// ask chatgpt

	callback := func(params *chatgpt.CallParams, err error) {
		if params == nil {
			return
		}
		if err != nil {
			log.Println(params.MsgId, err)
		}
		if params.Done {
			log.Println("answer: ", params.MsgId, params.Text)
		}else{
			log.Println("answer: ", params.MsgId, params.Chunk)
		}
	}
*/
func (chat *ChatGPTConversion) Ask(ctx context.Context, prompt string, callback func(answer *params.Answer, err error)) (err error) {
	defer func() {
		if err != nil && callback != nil {
			callback(nil, err)
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	msgId, parentId := chat.requst.PutUserMsg(prompt, "")
	msg := chat.requst.GetMessage()
	// log.Println("send message: ", msg)
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 1000,
		Messages:  msg,
		Stream:    true,
	}
	defer func() {
		if err != nil {
			chat.requst.PopMsg()
		}
	}()
	stream, err := chat.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return err
	}
	defer stream.Close()

	var text string
	var chunkIndex int
	for {
		chunkIndex += 1
		var response openai.ChatCompletionStreamResponse
		response, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			if callback != nil {
				callback(params.NewAnswer(msgId, parentId, "", text, true, chunkIndex), nil)
			}
			return nil
		}
		if err != nil {
			log.Println("stream error: ", err)
			if callback != nil {
				callback(params.NewAnswer(msgId, parentId, "", text, true, chunkIndex), err)
			}
			return
		}
		chunk := response.Choices[0].Delta.Content
		text += chunk
		if response.Choices[0].FinishReason != "" {
			chat.requst.SetResStream(msgId, text, &response)
			if callback != nil {
				callback(params.NewAnswer(msgId, parentId, chunk, text, true, chunkIndex), nil)
			}
			break
		}
		if callback != nil {
			callback(params.NewAnswer(msgId, parentId, chunk, text, false, chunkIndex), err)
		}
	}
	return nil
}

func (chat *ChatGPTConversion) RefreshProxy(proxy string) error {
	if chat.proxy == proxy {
		return nil
	}
	if proxy == "" {
		chat.botConfig.HTTPClient = &http.Client{}
		chat.proxy = proxy
		return nil
	}
	if err := chat.SetProxy(proxy); err != nil {
		return err
	}
	chat.client = openai.NewClientWithConfig(chat.botConfig)
	return nil
}

func (chat *ChatGPTConversion) RefreshSecretKey(secretKey string) error {
	if chat.secretKey == secretKey {
		return nil
	}
	chat.botConfig = openai.DefaultConfig(secretKey)
	err := chat.SetProxy(chat.proxy)
	if err != nil {
		return err
	}
	chat.client = openai.NewClientWithConfig(chat.botConfig)
	return nil
}
