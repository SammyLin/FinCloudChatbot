package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	
  _ "github.com/joho/godotenv/autoload"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

var (
	bot   *messaging_api.MessagingApiAPI
	debug bool
)

func main() {
	debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	var err error
	bot, err = messaging_api.NewMessagingApiAPI(
		os.Getenv("CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/callback", callbackHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	debugLog("Starting server on port " + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func callbackHandler(w http.ResponseWriter, req *http.Request) {
	debugLog("Received callback request")
	cb, err := webhook.ParseRequest(os.Getenv("CHANNEL_SECRET"), req)
	if err != nil {
		debugLog("Error parsing request: " + err.Error())
		if errors.Is(err, webhook.ErrInvalidSignature) {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range cb.Events {
		debugLog("Handling event: " + string(event.GetType()))
		switch e := event.(type) {
		case webhook.MessageEvent:
			handleMessageEvent(e)
		default:
			debugLog("Unsupported event type: " + string(event.GetType()))
		}
	}
}

func handleMessageEvent(event webhook.MessageEvent) {
	switch message := event.Message.(type) {
	case webhook.TextMessageContent:
		debugLog("Received text message: " + message.Text)
		if _, err := bot.ReplyMessage(
			&messaging_api.ReplyMessageRequest{
				ReplyToken: event.ReplyToken,
				Messages: []messaging_api.MessageInterface{
					messaging_api.TextMessage{
						Text: message.Text,
					},
				},
			},
		); err != nil {
			debugLog("Error replying to message: " + err.Error())
		} else {
			debugLog("Sent text reply")
		}
	default:
		debugLog("Unsupported message content: " + string(event.Message.GetType()))
	}
}

func debugLog(message string) {
	if debug {
		log.Println("[DEBUG] " + message)
	}
}
