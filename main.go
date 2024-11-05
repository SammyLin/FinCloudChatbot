package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

var (
	debugLogging bool
)

// CallbackConfig defines the configuration for each callback
type CallbackConfig struct {
	Type                 string `json:"Type"`
	API_AUTH_TOKEN       string `json:"API_AUTH_TOKEN,omitempty"`
	API_URL              string `json:"API_URL,omitempty"`
	CHANNEL_SECRET       string `json:"CHANNEL_SECRET"`
	CHANNEL_ACCESS_TOKEN string `json:"CHANNEL_ACCESS_TOKEN"`
}

// Configurations maps callback names to their configurations
type Configurations map[string]CallbackConfig

func main() {
	// Conditionally load .env only if not in production
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file:", err)
		}
	}

	// Set up logging
	log.SetOutput(os.Stdout)
	log.Println("Program started")

	// Load debug mode
	debugLogging, _ = strconv.ParseBool(os.Getenv("DEBUG_LOGGING"))
	if debugLogging {
		log.Println("Debug logging is enabled")
	} else {
		log.SetOutput(ioutil.Discard)
	}

	// Load configurations
	configs, err := loadConfigurations()
	if err != nil {
		log.Fatalf("Failed to load configurations: %v", err)
	}
	log.Printf("Loaded %d callback configurations", len(configs))

	// Register handlers for each callback
	for name, config := range configs {
		handler := createCallbackHandler(config, name)
		path := fmt.Sprintf("/%s/callback", name)
		http.HandleFunc(path, handler)
		debugLog("Registered callback handler at path: %s for %s", path, name)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
		debugLog("PORT not set, using default: 4000")
	} else {
		debugLog("Using PORT: %s", port)
	}

	debugLog("Server is starting, listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadConfigurations() (Configurations, error) {
	// Load configurations from CALLBACK_CONFIGS environment variable
	configsJSON := os.Getenv("CALLBACK_CONFIGS")
	if configsJSON == "" {
		return nil, errors.New("CALLBACK_CONFIGS is not set in the environment")
	}

	var configs Configurations
	err := json.Unmarshal([]byte(configsJSON), &configs)
	if err != nil {
		return nil, fmt.Errorf("error parsing CALLBACK_CONFIGS: %v", err)
	}

	return configs, nil
}

func createCallbackHandler(config CallbackConfig, name string) http.HandlerFunc {
	// Initialize Messaging API client
	bot, err := messaging_api.NewMessagingApiAPI(config.CHANNEL_ACCESS_TOKEN)
	if err != nil {
		log.Fatalf("Failed to initialize Messaging API for %s: %v", name, err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		debugLog("Received callback request for %s", name)

		cb, err := webhook.ParseRequest(config.CHANNEL_SECRET, r)
		if err != nil {
			debugLog("Error parsing request for %s: %v", name, err)
			if errors.Is(err, webhook.ErrInvalidSignature) {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		for _, event := range cb.Events {
			debugLog("Handling event: %s for %s", string(event.GetType()), name)
			switch e := event.(type) {
			case webhook.MessageEvent:
				handleMessageEvent(e, bot, config, name)
			default:
				debugLog("Unsupported event type: %s for %s", string(event.GetType()), name)
			}
		}
	}
}

func handleMessageEvent(event webhook.MessageEvent, bot *messaging_api.MessagingApiAPI, config CallbackConfig, name string) {
	switch message := event.Message.(type) {
	case webhook.TextMessageContent:
		debugLog("Received text message: %s for %s", message.Text, name)

		// Execute actions based on callback type
		switch config.Type {
		case "bypass":
			// Call API and get response
			response, err := queryAPI(config.API_URL, config.API_AUTH_TOKEN, message.Text)
			if err != nil {
				debugLog("Error querying API for %s: %v", name, err)
				sendReply(bot, event.ReplyToken, "Sorry, I encountered an error processing your request.", name)
				return
			}
			// Check if response is empty
			if response == "" {
				debugLog("Empty response from API for %s", name)
				sendReply(bot, event.ReplyToken, "Sorry, I couldn't generate a response.", name)
				return
			}
			sendReply(bot, event.ReplyToken, response, name)
		case "periodicsummary":
			// Execute periodic summary actions
			summary := "This is a periodic summary."
			sendReply(bot, event.ReplyToken, summary, name)
		default:
			debugLog("Unknown Type: %s for %s", config.Type, name)
		}
	default:
		debugLog("Unsupported message content: %s for %s", string(event.Message.GetType()), name)
	}
}

// queryAPI calls the external API with the user's message
func queryAPI(apiURL string, authToken string, message string) (string, error) {
	// Prepare request data
	requestBody, err := json.Marshal(map[string]string{
		"question": message,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	debugLog("API Response: %s", string(body))

	// Parse response
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %v, body: %s", err, string(body))
	}

	// Check if Text is empty
	if result.Text == "" {
		return "", fmt.Errorf("empty text in API response")
	}

	return result.Text, nil
}

// sendReply sends a reply message back to the user
func sendReply(bot *messaging_api.MessagingApiAPI, replyToken string, text string, name string) {
	_, err := bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text: text,
				},
			},
		},
	)
	if err != nil {
		debugLog("Error replying to message for %s: %v", name, err)
	} else {
		debugLog("Sent reply for %s", name)
	}
}

// debugLog logs messages if debugLogging is enabled
func debugLog(format string, v ...interface{}) {
	if debugLogging {
		log.Printf(format, v...)
	}
}
