package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
)

type OpenAIRequest struct {
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
}

type OpenAIResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	slackVerificationToken := os.Getenv("SLACK_VERIFICATION_TOKEN")
	azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureApiKey := os.Getenv("AZURE_API_KEY")

	http.HandleFunc("/slack/command", func(w http.ResponseWriter, r *http.Request) {
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if !s.ValidateToken(slackVerificationToken) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		openAIReq := OpenAIRequest{
			Prompt:    s.Text,
			MaxTokens: 50,
		}

		reqBody, err := json.Marshal(openAIReq)
		if err != nil {
			http.Error(w, "Failed to encode request", http.StatusInternalServerError)
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", azureEndpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", azureApiKey))

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to send request", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var openAIResp OpenAIResponse
		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			http.Error(w, "Failed to decode response", http.StatusInternalServerError)
			return
		}

		// Check if Choices slice is not empty
		if len(openAIResp.Choices) == 0 {
			http.Error(w, "No response from OpenAI", http.StatusInternalServerError)
			return
		}

		message := openAIResp.Choices[0].Text
		response := map[string]string{
			"response_type": "in_channel",
			"text":          message,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
