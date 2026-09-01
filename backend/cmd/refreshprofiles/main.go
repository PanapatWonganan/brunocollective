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

	var fixed int
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
			p, err := meta.GetProfile(conv.ExternalID)
			if err != nil {
				fmt.Printf("  #%d %s %s: %v\n", conv.ID, conv.Platform, conv.ExternalID, err)
				continue
			}
			switch {
			case p.Name != "":
				updates["display_name"] = p.Name
			case p.Username != "":
				updates["display_name"] = "@" + p.Username
			}
			if p.ProfilePic != "" {
				updates["avatar_url"] = p.ProfilePic
			}
		}
		if len(updates) == 0 {
			continue
		}
		database.DB.Model(&models.Conversation{}).Where("id = ?", conv.ID).Updates(updates)
		fixed++
		fmt.Printf("  #%d %s -> %v\n", conv.ID, conv.Platform, updates["display_name"])
	}
	fmt.Printf("updated %d conversations\n", fixed)
}
