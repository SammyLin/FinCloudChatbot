package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	client *linebot.Client
	err    error
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	// 建立客戶端
	client, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_ACCESS_TOKEN"))

	if err != nil {
		log.Println(err.Error())
	}

	http.HandleFunc("/callback", callbackHandler)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := client.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}

		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:

				if _, err = client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
					log.Println(err.Error())
				}
			}
		}
	}
}
