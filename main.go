package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"bytes"
	"encoding/json"
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

		// 檢查消息是否以 "/" 開頭
		if len(message.Text) > 0 && message.Text[0] == '/' {
			apiResponse, err := queryAPI(message.Text[1:]) // 去掉開頭的 "/"
			if err != nil {
				debugLog("Error querying API: " + err.Error())
				apiResponse = "Sorry, I couldn't process your request. Error: " + err.Error()
			}

			if _, err := bot.ReplyMessage(
				&messaging_api.ReplyMessageRequest{
					ReplyToken: event.ReplyToken,
					Messages: []messaging_api.MessageInterface{
						messaging_api.TextMessage{
							Text: apiResponse,
						},
					},
				},
			); err != nil {
				debugLog("Error replying to message: " + err.Error())
				apiResponse = "Sorry, I couldn't process your request. Error: " + err.Error()
			} else {
				debugLog("Sent API response as reply")
			}
		} else {
			debugLog("Message ignored, not starting with '/'")
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

func queryAPI(question string) (string, error) {
	payload := map[string]string{
		"question": question,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling payload: %v", err)
	}

	req, err := http.NewRequest("POST", os.Getenv("API_URL"), bytes.NewBuffer(jsonPayload)) // 使用環境變數
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer " + os.Getenv("API_AUTH_TOKEN")) // 使用環境變數
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	debugLog("API Response: " + string(body))

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	debugLog("Parsed API Response: " + fmt.Sprintf("%+v", result))

	// 檢查 API 響應的 success 字段
	if success, ok := result["success"].(bool); ok && !success {
		if message, ok := result["message"].(string); ok {
			return message, nil // 返回 API 返回的錯誤訊息
		}
		return "An error occurred, but no message was provided.", nil
	}

	// 檢查所有可能的字段名
	possibleFields := []string{"answer", "response", "result", "output", "text"}
	for _, field := range possibleFields {
		if answer, ok := result[field]; ok {
			if str, ok := answer.(string); ok {
				return str, nil
			}
			return fmt.Sprintf("%v", answer), nil
		}
	}

	return "", errors.New("unexpected API response format")
}
