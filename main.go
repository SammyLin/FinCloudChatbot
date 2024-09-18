package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

var (
	client *linebot.Client
	err    error
	debug  bool
)

func main() {
	// Initialize debug mode
	debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	debugLog("Using port: " + port)

	// Create LINE bot client
	client, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_ACCESS_TOKEN"))
	if err != nil {
		log.Println(err.Error())
	}
	debugLog("LINE Bot client created")

	http.HandleFunc("/callback", callbackHandler)
	debugLog("Callback handler set")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	debugLog("Received callback request")
	debugLog("HTTP Method: " + r.Method)

	// Log request headers
	debugLog("Request Headers:")
	for name, values := range r.Header {
		for _, value := range values {
			debugLog(name + ": " + value)
		}
	}

	// Log request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		debugLog("Error reading request body: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	debugLog("Request Body: " + string(body))

	// Validate signature
	events, err := client.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			debugLog("Invalid signature")
			w.WriteHeader(http.StatusBadRequest)
		} else {
			debugLog("Error parsing request: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// Process events
	for _, event := range events {
		if err := handleEvent(event); err != nil {
			debugLog("Error handling event: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func handleEvent(event *linebot.Event) error {
	// 在這裡處理事件
	switch event.Type {
	case linebot.EventTypeMessage:
		switch message := event.Message.(type) {
		case *linebot.TextMessage:
			debugLog("Received text message: " + message.Text)
			if _, err := client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
				return err
			}
		}
	}
	return nil
}

func debugLog(message string) {
	if debug {
		log.Println("[DEBUG] " + message)
	}
}
