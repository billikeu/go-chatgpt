package main

import (
	"context"
	"log"
	"time"

	"github.com/billikeu/go-chatgpt/chatgpt"
	"github.com/billikeu/go-chatgpt/chatgptuno"
	"github.com/billikeu/go-chatgpt/params"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// ExampleChatGPTUno()
	ExampleChatGPT()
	// wait for change title
	time.Sleep(time.Second * 10)
}

func ExampleChatGPTUno() {
	var conversationId string
	var parentId string

	// Every time data is received, this callback will be called and executed sequentially
	callback := func(chatRes *chatgptuno.Response, err error) {
		/*
		   Support churn transmission, here we only print the last time
		*/
		if err != nil {
			log.Println(err)
			return
		}
		if chatRes.Message.Metadata.FinishDetails.Stop != "" {
			log.Println(chatRes.ConversationID, chatRes.Message.ID, chatRes.Message.Content.Parts)
			conversationId = chatRes.ConversationID
			parentId = chatRes.Message.ID
		}
	}

	chat := chatgptuno.NewChatGPTUnoBot(&chatgptuno.ChatGPTUnoConfig{
		Model:       "text-davinci-002-render-sha",
		Proxy:       "", // http://127.0.0.1:9080
		AccessToken: "your access token",
	})
	if err := chat.Ask("give me a joke.", conversationId, parentId, "", 360, callback); err != nil {
		panic(err)
	}
	if err := chat.Ask("it's not funny.", conversationId, parentId, "", 360, callback); err != nil {
		panic(err)
	}
	log.Println(conversationId, parentId)
}

func ExampleChatGPT() error {
	secretKey := ""
	proxy := ""
	systemRoleMessage := ""

	conversation := chatgpt.NewChatGPTConversion(secretKey)
	err := conversation.SetProxy(proxy)
	if err != nil {
		return err
	}
	err = conversation.Init()
	if err != nil {
		return err
	}
	if systemRoleMessage != "" {
		conversation.SetSystemMsg(systemRoleMessage)
	}
	callback := func(answer *params.Answer, err error) {
		if err != nil {
			log.Println(err)
			return
		}
		if answer == nil {
			return
		}
		log.Println(answer.Done, answer.MsgId, answer.Text)
	}
	err = conversation.Ask(context.Background(), "tell me a joke", callback)
	if err != nil {
		return err
	}
	err = conversation.Ask(context.Background(), "it's not funny", callback)
	if err != nil {
		return err
	}
	return nil
}
