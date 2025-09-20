package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

type Guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Channel struct {
	ID                         string  `json:"id"`
	Name                       string  `json:"name"`
	Type                       int     `json:"type"`
	GuildID                    string  `json:"guild_id"`
	Position                   int     `json:"position"`
	Topic                      string  `json:"topic,omitempty"`
	ParentID                   *string `json:"parent_id,omitempty"`
	NSFW                       bool    `json:"nsfw,omitempty"`
	Bitrate                    int     `json:"bitrate,omitempty"`
	UserLimit                  int     `json:"user_limit,omitempty"`
	RateLimitPerUser           int     `json:"rate_limit_per_user,omitempty"`
	DefaultAutoArchiveDuration int     `json:"default_auto_archive_duration,omitempty"`
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

type Webhook struct {
	ID        string  `json:"id"`
	Type      int     `json:"type"`
	GuildID   string  `json:"guild_id,omitempty"`
	ChannelID string  `json:"channel_id"`
	Name      string  `json:"name"`
	Avatar    *string `json:"avatar"`
	Token     string  `json:"token,omitempty"`
	URL       string  `json:"url,omitempty"`
}

type Invite struct {
	Code      string `json:"code"`
	Guild     *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"guild,omitempty"`
	Channel *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type int    `json:"type"`
	} `json:"channel,omitempty"`
	Inviter *struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"inviter,omitempty"`
	MaxAge    int `json:"max_age"`
	MaxUses   int `json:"max_uses"`
	Temporary bool `json:"temporary"`
	CreatedAt string `json:"created_at"`
	Uses      int    `json:"uses"`
}

func main() {
	// CLI flags
	var (
		guildFlag       = flag.String("guild", "", "Specific guild ID to introspect (optional)")
		outputDir       = flag.String("output", "discord-resources", "Output directory for generated manifests")
		includeRoles    = flag.Bool("roles", true, "Include roles in introspection")
		includeChannels = flag.Bool("channels", true, "Include channels in introspection")
		includeGuilds   = flag.Bool("guilds", true, "Include guilds in introspection")
		includeWebhooks = flag.Bool("webhooks", true, "Include webhooks in introspection (future provider support)")
		includeInvites  = flag.Bool("invites", true, "Include invites in introspection (future provider support)")
		discoveryMode   = flag.Bool("discovery", false, "Discovery mode: generate YAML even for unsupported resources")
	)
	flag.Parse()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN environment variable not set")
	}

	// Get all guilds the bot is a member of
	guilds := getGuilds(token)
	fmt.Printf("Found %d guilds\n", len(guilds))

	// Filter by specific guild if requested
	if *guildFlag != "" {
		filtered := []Guild{}
		for _, guild := range guilds {
			if guild.ID == *guildFlag {
				filtered = append(filtered, guild)
				break
			}
		}
		guilds = filtered
		if len(guilds) == 0 {
			log.Fatalf("Guild with ID %s not found or bot not a member", *guildFlag)
		}
	}

	// Create output directory
	os.MkdirAll(*outputDir, 0755)

	for _, guild := range guilds {
		fmt.Printf("Processing guild: %s (%s)\n", guild.Name, guild.ID)

		// Generate Guild CR
		if *includeGuilds {
			guildCR := generateGuildCR(guild)
			writeFile(fmt.Sprintf("%s/guild-%s.yaml", *outputDir, sanitizeName(guild.Name)), guildCR)
		}

		// Get channels for this guild with proper ordering
		if *includeChannels {
			channels := getChannels(token, guild.ID)
			generateChannelManifests(channels, guild.Name, *outputDir)
		}

		// Get roles for this guild
		if *includeRoles {
			roles := getRoles(token, guild.ID)
			for _, role := range roles {
				if role.Managed || role.Name == "@everyone" {
					continue // Skip managed and default roles
				}
				roleCR := generateRoleCR(role, guild.Name, guild.ID)
				writeFile(fmt.Sprintf("%s/role-%s-%s.yaml", *outputDir, sanitizeName(guild.Name), sanitizeName(role.Name)), roleCR)
			}
		}

		// Get webhooks for this guild
		if *includeWebhooks && (*discoveryMode || checkProviderSupport("webhooks")) {
			webhooks := getWebhooks(token, guild.ID)
			for _, webhook := range webhooks {
				webhookCR := generateWebhookCR(webhook, guild.Name, *discoveryMode)
				writeFile(fmt.Sprintf("%s/webhook-%s-%s.yaml", *outputDir, sanitizeName(guild.Name), sanitizeName(webhook.Name)), webhookCR)
			}
		}

		// Get invites for this guild
		if *includeInvites && (*discoveryMode || checkProviderSupport("invites")) {
			invites := getInvites(token, guild.ID)
			for _, invite := range invites {
				inviteCR := generateInviteCR(invite, guild.Name, *discoveryMode)
				writeFile(fmt.Sprintf("%s/invite-%s-%s.yaml", *outputDir, sanitizeName(guild.Name), invite.Code), inviteCR)
			}
		}
	}

	fmt.Printf("Resource generation complete! Check %s/ directory\n", *outputDir)
	if *discoveryMode {
		fmt.Println("Note: Discovery mode enabled - all Discord resources discovered")
	}
	fmt.Println("✅ Supported: Guilds, Channels, Roles, Webhooks, Invites")
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

func getWebhooks(token, guildID string) []Webhook {
	resp := makeRequest("GET", fmt.Sprintf("https://discord.com/api/v10/guilds/%s/webhooks", guildID), token)
	defer resp.Body.Close()

	var webhooks []Webhook
	if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
		log.Printf("Warning: Failed to decode webhooks for guild %s: %v", guildID, err)
		return []Webhook{}
	}
	return webhooks
}

func getInvites(token, guildID string) []Invite {
	resp := makeRequest("GET", fmt.Sprintf("https://discord.com/api/v10/guilds/%s/invites", guildID), token)
	defer resp.Body.Close()

	var invites []Invite
	if err := json.NewDecoder(resp.Body).Decode(&invites); err != nil {
		log.Printf("Warning: Failed to decode invites for guild %s: %v", guildID, err)
		return []Invite{}
	}
	return invites
}

func checkProviderSupport(resourceType string) bool {
	// Supported Discord resources by provider-discord
	supportedResources := map[string]bool{
		"guilds":   true,
		"channels": true,
		"roles":    true,
		"webhooks": true,  // Now supported!
		"invites":  true,  // Now supported!
	}
	return supportedResources[resourceType]
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
	return fmt.Sprintf(`apiVersion: guild.discord.crossplane.io/v1alpha1
kind: Guild
metadata:
  name: %s
  annotations:
    discord.crossplane.io/id: "%s"
spec:
  forProvider:
    name: "%s"
  providerConfigRef:
    name: discord-provider-config
`, sanitizeName(guild.Name), guild.ID, guild.Name)
}

// generateChannelManifests creates channel manifests with proper dependency ordering
func generateChannelManifests(channels []Channel, guildName, outputDir string) {
	// Separate categories from regular channels
	categories := []Channel{}
	regularChannels := []Channel{}

	for _, channel := range channels {
		if channel.Type == 4 { // Category
			categories = append(categories, channel)
		} else {
			regularChannels = append(regularChannels, channel)
		}
	}

	// Sort categories by position
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Position < categories[j].Position
	})

	// Sort regular channels by position
	sort.Slice(regularChannels, func(i, j int) bool {
		return regularChannels[i].Position < regularChannels[j].Position
	})

	// Create category manifests first
	for _, category := range categories {
		channelCR := generateChannelCR(category, guildName)
		writeFile(fmt.Sprintf("%s/channel-%s-%s.yaml", outputDir, sanitizeName(guildName), sanitizeName(category.Name)), channelCR)
	}

	// Create regular channel manifests
	for _, channel := range regularChannels {
		channelCR := generateChannelCR(channel, guildName)
		writeFile(fmt.Sprintf("%s/channel-%s-%s.yaml", outputDir, sanitizeName(guildName), sanitizeName(channel.Name)), channelCR)
	}
}

func generateChannelCR(channel Channel, guildName string) string {
	channelTypeName := getChannelTypeName(channel.Type)

	cr := fmt.Sprintf(`apiVersion: channel.discord.crossplane.io/v1alpha1
kind: Channel
metadata:
  name: %s-%s
  annotations:
    discord.crossplane.io/id: "%s"
    discord.crossplane.io/type: "%s"
spec:
  forProvider:
    name: "%s"
    type: %d
    guildId: "%s"
    position: %d`,
		sanitizeName(guildName), sanitizeName(channel.Name),
		channel.ID, channelTypeName, channel.Name, channel.Type,
		channel.GuildID, channel.Position)

	// Add parent_id for channels under categories
	if channel.ParentID != nil && *channel.ParentID != "" {
		cr += fmt.Sprintf(`
    parentId: "%s"`, *channel.ParentID)
	}

	// Add optional fields based on channel type
	if channel.Topic != "" && (channel.Type == 0 || channel.Type == 5) { // Text or News channels
		cr += fmt.Sprintf(`
    topic: "%s"`, strings.ReplaceAll(channel.Topic, `"`, `\"`))
	}

	if channel.Type == 0 || channel.Type == 5 { // Text or News channels
		if channel.NSFW {
			cr += `
    nsfw: true`
		}
		if channel.RateLimitPerUser > 0 {
			cr += fmt.Sprintf(`
    rateLimitPerUser: %d`, channel.RateLimitPerUser)
		}
		if channel.DefaultAutoArchiveDuration > 0 {
			cr += fmt.Sprintf(`
    defaultAutoArchiveDuration: %d`, channel.DefaultAutoArchiveDuration)
		}
	}

	if channel.Type == 2 || channel.Type == 13 { // Voice or Stage channels
		if channel.Bitrate > 0 {
			cr += fmt.Sprintf(`
    bitrate: %d`, channel.Bitrate)
		}
		if channel.UserLimit > 0 {
			cr += fmt.Sprintf(`
    userLimit: %d`, channel.UserLimit)
		}
	}

	cr += `
  providerConfigRef:
    name: discord-provider-config
`
	return cr
}

func getChannelTypeName(channelType int) string {
	switch channelType {
	case 0:
		return "text"
	case 2:
		return "voice"
	case 4:
		return "category"
	case 5:
		return "news"
	case 13:
		return "stage-voice"
	case 15:
		return "forum"
	default:
		return fmt.Sprintf("type-%d", channelType)
	}
}

func generateRoleCR(role Role, guildName string, guildID string) string {
	return fmt.Sprintf(`apiVersion: role.discord.crossplane.io/v1alpha1
kind: Role
metadata:
  name: %s-%s
  annotations:
    discord.crossplane.io/id: "%s"
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

func generateWebhookCR(webhook Webhook, guildName string, discoveryMode bool) string {
	comment := ""
	if !checkProviderSupport("webhooks") && discoveryMode {
		comment = `# NOTE: Webhooks require provider-discord v0.4.0+
# This manifest is ready for deployment
# `
	}

	return fmt.Sprintf(`%sapiVersion: webhook.discord.crossplane.io/v1alpha1
kind: Webhook
metadata:
  name: %s-%s
  annotations:
    discord.crossplane.io/id: "%s"
    discord.crossplane.io/type: "%s"
spec:
  forProvider:
    name: "%s"
    channelId: "%s"
    guildId: "%s"
  providerConfigRef:
    name: discord-provider-config
`, comment, sanitizeName(guildName), sanitizeName(webhook.Name),
		webhook.ID, getWebhookTypeName(webhook.Type), webhook.Name,
		webhook.ChannelID, webhook.GuildID)
}

func generateInviteCR(invite Invite, guildName string, discoveryMode bool) string {
	comment := ""
	if !checkProviderSupport("invites") && discoveryMode {
		comment = `# NOTE: Invites require provider-discord v0.4.0+
# This manifest is ready for deployment
# `
	}

	channelName := "unknown-channel"
	channelID := ""
	if invite.Channel != nil {
		channelName = invite.Channel.Name
		channelID = invite.Channel.ID
	}

	return fmt.Sprintf(`%sapiVersion: invite.discord.crossplane.io/v1alpha1
kind: Invite
metadata:
  name: %s-%s
  annotations:
    discord.crossplane.io/code: "%s"
    discord.crossplane.io/channel: "%s"
    discord.crossplane.io/created-at: "%s"
    discord.crossplane.io/uses: "%d"
spec:
  forProvider:
    code: "%s"
    channelId: "%s"
    maxAge: %d
    maxUses: %d
    temporary: %t
  providerConfigRef:
    name: discord-provider-config
`, comment, sanitizeName(guildName), sanitizeName(channelName),
		invite.Code, channelName, invite.CreatedAt, invite.Uses,
		invite.Code, channelID, invite.MaxAge, invite.MaxUses, invite.Temporary)
}

func getWebhookTypeName(webhookType int) string {
	switch webhookType {
	case 1:
		return "incoming"
	case 2:
		return "channel-follower"
	case 3:
		return "application"
	default:
		return fmt.Sprintf("type-%d", webhookType)
	}
}

func writeFile(filename, content string) {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing %s: %v", filename, err)
	} else {
		fmt.Printf("  Created: %s\n", filename)
	}
}
