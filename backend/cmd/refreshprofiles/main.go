// refreshprofiles re-fetches chat display names/avatars for conversations
// still showing a placeholder ("Facebook User", "Instagram User", "LINE
// User"). Run it once after the Meta app goes Live — profile lookups fail
// while the app is in Development Mode, and first-contact fetches that failed
// left threads unnamed.
//
//	cd /opt/inventory/backend && go run ./cmd/refreshprofiles
package main

import (
	"fmt"

	"brunocollective_inventory/config"
	"brunocollective_inventory/database"
	"brunocollective_inventory/models"
	"brunocollective_inventory/services"
)

func main() {
	cfg := config.Load()
	database.Connect(cfg)
	line := services.NewLineClient(cfg)
	meta := services.NewMetaClient(cfg)

	var convs []models.Conversation
	database.DB.
		Where("display_name IN ? OR display_name = ''", []string{"LINE User", "Facebook User", "Instagram User"}).
		Find(&convs)
	fmt.Printf("%d conversations with placeholder names\n", len(convs))

	// Bulk name map from the page conversations endpoint — a handful of
	// paged requests instead of one profile call per thread.
	names := map[string]string{}
	for _, platform := range []string{"", "instagram"} {
		m, err := meta.ListConversationNames(platform)
		if err != nil {
			fmt.Printf("list conversations (%q): %v (got %d so far)\n", platform, err, len(m))
		}
		for id, name := range m {
			names[id] = name
		}
	}
	fmt.Printf("%d participant names from the conversations API\n", len(names))

	var fixed, missed int
	for _, conv := range convs {
		updates := map[string]interface{}{}
		switch conv.Platform {
		case "line":
			if p, err := line.GetProfile(conv.ExternalID); err == nil && p.DisplayName != "" {
				updates["display_name"] = p.DisplayName
				if p.PictureURL != "" {
					updates["avatar_url"] = p.PictureURL
				}
			}
		case "facebook", "instagram":
			if name := names[conv.ExternalID]; name != "" {
				updates["display_name"] = name
			}
		}
		if len(updates) == 0 {
			missed++
			continue
		}
		database.DB.Model(&models.Conversation{}).Where("id = ?", conv.ID).Updates(updates)
		fixed++
	}
	fmt.Printf("updated %d conversations, %d unresolved\n", fixed, missed)
}
