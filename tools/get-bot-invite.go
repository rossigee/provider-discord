package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Application struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable not set")
	}

	// Get application info
	req, _ := http.NewRequest("GET", "https://discord.com/api/v10/oauth2/applications/@me", nil)
	req.Header.Set("Authorization", "Bot "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Discord API error: %d - %s", resp.StatusCode, string(body))
	}

	var app Application
	json.NewDecoder(resp.Body).Decode(&app)

	fmt.Printf("Bot Name: %s\n", app.Name)
	fmt.Printf("Application ID: %s\n\n", app.ID)

	// Generate invite link with admin permissions
	inviteURL := fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&permissions=8&scope=bot", app.ID)

	fmt.Println("Bot Invite Link (with Administrator permissions):")
	fmt.Println(inviteURL)
	fmt.Println("\nOpen this link in your browser to add the bot to your server!")
}
