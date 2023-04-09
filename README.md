# go-chatgpt
go-chatgpt: Reverse engineered API of OpenAi's ChatGPT. ChatGPT聊天功能的逆向工程

go-chatgpt is a ChatGPT unofficial API developed using Golang. With most chatbot APIs being built on Python, go-chatgpt is unique in its ability to be easily compiled and deployed. It's designed to work seamlessly with your current applications, providing a customizable and reliable chatbot experience.

## Setup

```
go get -u github.com/billikeu/go-chatgpt/chatgptuno
```

## Example bot

```golang
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
		Proxy:       "http://127.0.0.1:9080",
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
```

- Set environment variable `CHATGPT_BASE_URL` to change BaseURL, default: `https://bypass.churchless.tech/api/`

## Others

- https://github.com/billikeu/Go-EdgeGPT
- https://github.com/billikeu/Go-ChatBot
- https://github.com/billikeu/ChatGPT-App
- https://github.com/billikeu/AIArchive
- https://github.com/billikeu/go-chatgpt


## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=billikeu/go-chatgpt&type=Date)](https://star-history.com/#billikeu/go-chatgpt&Date)

## Contributors

This project exists thanks to all the people who contribute.

 <a href="github.com/billikeu/go-chatgpt/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=billikeu/go-chatgpt" />
 </a>

## Reference
