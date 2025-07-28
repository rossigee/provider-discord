package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Channel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	GuildID  string `json:"guild_id"`
	Position int    `json:"position"`
	Topic    string `json:"topic,omitempty"`
}

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Position    int    `json:"position"`
	Permissions string `json:"permissions"`
	Color       int    `json:"color"`
	Hoist       bool   `json:"hoist"`
	Managed     bool   `json:"managed"`
	Mentionable bool   `json:"mentionable"`
}

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable not set")
	}

	// Get all guilds the bot is a member of
	guilds := getGuilds(token)
	fmt.Printf("Found %d guilds\n", len(guilds))
	
	// Create output directory
	os.MkdirAll("discord-resources", 0755)
	
	for _, guild := range guilds {
		fmt.Printf("Processing guild: %s (%s)\n", guild.Name, guild.ID)
		
		// Generate Guild CR
		guildCR := generateGuildCR(guild)
		writeFile(fmt.Sprintf("discord-resources/guild-%s.yaml", sanitizeName(guild.Name)), guildCR)
		
		// Get channels for this guild
		channels := getChannels(token, guild.ID)
		for _, channel := range channels {
			channelCR := generateChannelCR(channel, guild.Name)
			writeFile(fmt.Sprintf("discord-resources/channel-%s-%s.yaml", sanitizeName(guild.Name), sanitizeName(channel.Name)), channelCR)
		}
		
		// Get roles for this guild  
		roles := getRoles(token, guild.ID)
		for _, role := range roles {
			if role.Managed || role.Name == "@everyone" {
				continue // Skip managed and default roles
			}
			roleCR := generateRoleCR(role, guild.Name, guild.ID)
			writeFile(fmt.Sprintf("discord-resources/role-%s-%s.yaml", sanitizeName(guild.Name), sanitizeName(role.Name)), roleCR)
		}
	}
	
	fmt.Println("Resource generation complete! Check discord-resources/ directory")
}

func getGuilds(token string) []Guild {
	resp := makeRequest("GET", "https://discord.com/api/v10/users/@me/guilds", token)
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Guilds response: %s\n", string(body))
	
	var guilds []Guild
	json.Unmarshal(body, &guilds)
	return guilds
}

func getChannels(token, guildID string) []Channel {
	resp := makeRequest("GET", fmt.Sprintf("https://discord.com/api/v10/guilds/%s/channels", guildID), token)
	defer resp.Body.Close()
	
	var channels []Channel
	json.NewDecoder(resp.Body).Decode(&channels)
	return channels
}

func getRoles(token, guildID string) []Role {
	resp := makeRequest("GET", fmt.Sprintf("https://discord.com/api/v10/guilds/%s/roles", guildID), token)
	defer resp.Body.Close()
	
	var roles []Role
	json.NewDecoder(resp.Body).Decode(&roles)
	return roles
}

func makeRequest(method, url, token string) *http.Response {
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", "Bot "+token)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Discord API error: %d - %s", resp.StatusCode, string(body))
	}
	
	return resp
}

func generateGuildCR(guild Guild) string {
	return fmt.Sprintf(`apiVersion: guild.discord.golder.tech/v1alpha1
kind: Guild
metadata:
  name: %s
  annotations:
    discord.golder.tech/id: "%s"
spec:
  forProvider:
    name: "%s"
  providerConfigRef:
    name: discord-provider-config
`, sanitizeName(guild.Name), guild.ID, guild.Name)
}

func generateChannelCR(channel Channel, guildName string) string {
	cr := fmt.Sprintf(`apiVersion: channel.discord.golder.tech/v1alpha1
kind: Channel
metadata:
  name: %s-%s
  annotations:
    discord.golder.tech/id: "%s"
spec:
  forProvider:
    name: "%s"
    type: %d
    guildId: "%s"
    position: %d`,
		sanitizeName(guildName), sanitizeName(channel.Name),
		channel.ID, channel.Name, channel.Type,
		channel.GuildID, channel.Position)
	
	if channel.Topic != "" {
		cr += fmt.Sprintf(`
    topic: "%s"`, channel.Topic)
	}
	
	cr += `
  providerConfigRef:
    name: discord-provider-config
`
	return cr
}

func generateRoleCR(role Role, guildName string, guildID string) string {
	return fmt.Sprintf(`apiVersion: role.discord.golder.tech/v1alpha1
kind: Role
metadata:
  name: %s-%s
  annotations:
    discord.golder.tech/id: "%s"
spec:
  forProvider:
    name: "%s"
    guildId: "%s"
    color: %d
    hoist: %t
    mentionable: %t
    permissions: "%s"
    position: %d
  providerConfigRef:
    name: discord-provider-config
`, sanitizeName(guildName), sanitizeName(role.Name), role.ID,
		role.Name, guildID, role.Color, 
		role.Hoist, role.Mentionable, role.Permissions, role.Position)
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ".", "-")
	
	// Remove non-ASCII characters for Kubernetes compliance
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result += string(r)
		}
	}
	
	// Ensure it doesn't start or end with hyphen
	result = strings.Trim(result, "-")
	if result == "" {
		result = "unnamed"
	}
	
	return result
}

func writeFile(filename, content string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing %s: %v", filename, err)
	} else {
		fmt.Printf("  Created: %s\n", filename)
	}
}