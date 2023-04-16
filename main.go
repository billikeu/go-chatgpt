package main

import (
	"log"
	"time"

	"github.com/billikeu/go-chatgpt/chatgptuno"
)

var conversationId string
var parentId string

// Every time data is received, this callback will be called and executed sequentially
func callback(chatRes *chatgptuno.Response, err error) {
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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
	// wait for change title
	time.Sleep(time.Second * 10)
}
