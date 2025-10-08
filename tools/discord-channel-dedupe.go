package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type DuplicateGroup struct {
	Name      string
	Channels  []Channel
	KeepIndex int // Index of the channel to keep (oldest by position)
}

func main() {
	// CLI flags
	var (
		guildFlag  = flag.String("guild", "", "Specific guild ID to analyze")
		dryRun     = flag.Bool("dry-run", false, "Force dry run mode (default behavior)")
		confirm    = flag.Bool("confirm", false, "Actually perform deletions (DANGER)")
		outputFile = flag.String("output", "", "Output file for deletion plan (optional)")
	)
	flag.Parse()

	// Default behavior is dry run unless explicitly confirmed
	isDryRun := !*confirm
	if *dryRun && *confirm {
		log.Fatal("Cannot use both --dry-run and --confirm")
	}

	token := os.Getenv("DISCORD_BOT_TOKEN")

	// Debug: Show the mode we're running in
	if isDryRun {
		fmt.Println("🔍 Running in DRY RUN mode (safe - no changes will be made)")
	} else {
		fmt.Println("🗑️  Running in DELETION mode (dangerous - channels will be deleted)")
	}

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

	for _, guild := range guilds {
		fmt.Printf("\n🔍 Analyzing guild: %s (%s)\n", guild.Name, guild.ID)
		analyzeAndDedupeChannels(token, guild, isDryRun, *confirm, *outputFile)
	}
}

func analyzeAndDedupeChannels(token string, guild Guild, dryRun, confirm bool, outputFile string) {
	channels := getChannels(token, guild.ID)

	if len(channels) == 0 {
		fmt.Printf("  No channels found in guild\n")
		return
	}

	fmt.Printf("  Found %d total channels\n", len(channels))

	// Group channels by name to find duplicates
	nameGroups := make(map[string][]Channel)
	for _, channel := range channels {
		nameGroups[channel.Name] = append(nameGroups[channel.Name], channel)
	}

	// Find duplicate groups
	var duplicates []DuplicateGroup
	totalDuplicates := 0

	for name, group := range nameGroups {
		if len(group) > 1 {
			// Find the channel to keep (lowest position = oldest/highest priority)
			keepIndex := 0
			minPosition := group[0].Position

			for i, channel := range group {
				if channel.Position < minPosition {
					minPosition = channel.Position
					keepIndex = i
				}
			}

			duplicates = append(duplicates, DuplicateGroup{
				Name:      name,
				Channels:  group,
				KeepIndex: keepIndex,
			})
			totalDuplicates += len(group) - 1
		}
	}

	if len(duplicates) == 0 {
		fmt.Printf("  ✅ No duplicate channels found!\n")
		return
	}

	fmt.Printf("  ⚠️  Found %d duplicate channel groups with %d total duplicate channels\n", len(duplicates), totalDuplicates)

	// Output analysis
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Channel Deduplication Plan for Guild: %s (%s)\n", guild.Name, guild.ID))
	output.WriteString(fmt.Sprintf("# Generated at: %s\n\n", "now"))
	output.WriteString(fmt.Sprintf("# Total channels: %d\n", len(channels)))
	output.WriteString(fmt.Sprintf("# Duplicate groups: %d\n", len(duplicates)))
	output.WriteString(fmt.Sprintf("# Channels to delete: %d\n\n", totalDuplicates))

	for _, dup := range duplicates {
		output.WriteString(fmt.Sprintf("## Duplicate Group: '%s'\n", dup.Name))
		output.WriteString(fmt.Sprintf("# Found %d channels with this name\n\n", len(dup.Channels)))

		for i, channel := range dup.Channels {
			action := "DELETE ❌"
			if i == dup.KeepIndex {
				action = "KEEP ✅ (oldest position)"
			}

			output.WriteString(fmt.Sprintf("- %s Channel ID: %s, Position: %d, Type: %s\n",
				action, channel.ID, channel.Position, getChannelTypeName(channel.Type)))
		}
		output.WriteString("\n")
	}

	// Print to console
	fmt.Print(output.String())

	// Write to file if requested
	if outputFile != "" {
		err := os.WriteFile(outputFile, []byte(output.String()), 0644)
		if err != nil {
			log.Printf("Error writing to file %s: %v", outputFile, err)
		} else {
			fmt.Printf("📝 Plan written to: %s\n", outputFile)
		}
	}

	// Perform deletions if confirmed
	if confirm {
		fmt.Printf("\n🗑️  PERFORMING ACTUAL DELETIONS...\n")
		deletedCount := 0

		for _, dup := range duplicates {
			for i, channel := range dup.Channels {
				if i != dup.KeepIndex {
					fmt.Printf("  Deleting channel '%s' (ID: %s)... ", channel.Name, channel.ID)
					err := deleteChannel(token, channel.ID)
					if err != nil {
						fmt.Printf("FAILED: %v\n", err)
					} else {
						fmt.Printf("SUCCESS\n")
						deletedCount++
					}
				}
			}
		}

		fmt.Printf("\n✅ Deleted %d duplicate channels\n", deletedCount)
	} else if !dryRun {
		fmt.Printf("\n⚠️  Use --confirm to actually perform deletions\n")
	}
}

func deleteChannel(token, channelID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://discord.com/api/v10/channels/%s", channelID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Discord API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func getGuilds(token string) []Guild {
	resp := makeRequest("GET", "https://discord.com/api/v10/users/@me/guilds", token)
	defer resp.Body.Close()

	var guilds []Guild
	json.NewDecoder(resp.Body).Decode(&guilds)
	return guilds
}

func getChannels(token, guildID string) []Channel {
	resp := makeRequest("GET", fmt.Sprintf("https://discord.com/api/v10/guilds/%s/channels", guildID), token)
	defer resp.Body.Close()

	var channels []Channel
	json.NewDecoder(resp.Body).Decode(&channels)
	return channels
}

func makeRequest(method, url, token string) *http.Response {
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", "Bot "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Discord API error: %d - %s", resp.StatusCode, string(body))
	}

	return resp
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
