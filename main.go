package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type SlackRequest struct {
	Token       string `form:"token"`
	TeamID      string `form:"team_id"`
	TeamDomain  string `form:"team_domain"`
	ChannelID   string `form:"channel_id"`
	ChannelName string `form:"channel_name"`
	UserID      string `form:"user_id"`
	UserName    string `form:"user_name"`
	Command     string `form:"command"`
	Text        string `form:"text"`
	ResponseURL string `form:"response_url"`
	TriggerID   string `form:"trigger_id"`
}

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

	azureEndpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	azureApiKey := os.Getenv("AZURE_API_KEY")
	slackToken := os.Getenv("SLACK_VERIFICATION_TOKEN")

	r := gin.Default()

	r.POST("/slack/command", func(c *gin.Context) {
		var slackReq SlackRequest
		if err := c.ShouldBind(&slackReq); err != nil {
			log.Printf("Bind error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if slackReq.Token != slackToken {
			log.Printf("Invalid token: %s", slackReq.Token)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		openAIReq := OpenAIRequest{
			Prompt:    slackReq.Text,
			MaxTokens: 50,
		}

		reqBody, err := json.Marshal(openAIReq)
		if err != nil {
			log.Printf("JSON marshal error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode request"})
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", azureEndpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			log.Printf("Request creation error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", azureApiKey))

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Request error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send request"})
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Read error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
			return
		}

		var openAIResp OpenAIResponse
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			log.Printf("Unmarshal error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
			return
		}

		message := openAIResp.Choices[0].Text
		c.JSON(http.StatusOK, gin.H{
			"response_type": "in_channel",
			"text":          message,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
