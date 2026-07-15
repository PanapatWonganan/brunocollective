package models

import "time"

// AutoReplyRule is one keyword-triggered action set for comments on
// Facebook posts / Instagram media. The first enabled rule (by priority,
// then id) whose keywords match the comment wins — one rule per comment.
type AutoReplyRule struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"not null"`
	// Platform: "facebook", "instagram", or "all".
	Platform string `json:"platform" gorm:"default:all"`
	// Keywords match case-insensitively as substrings; any hit triggers.
	// Empty list = match every comment (catch-all rule).
	Keywords StringSlice `json:"keywords" gorm:"type:text"`
	Enabled  bool        `json:"enabled" gorm:"default:true"`
	// Priority: lower runs first (ties broken by id).
	Priority int `json:"priority" gorm:"default:100"`

	// Actions — any combination. Texts support {name} = commenter name.
	ReplyText        string `json:"reply_text"`         // public reply under the comment
	PrivateReplyText string `json:"private_reply_text"` // DM to the commenter (one shot per comment)
	HideComment      bool   `json:"hide_comment"`       // hide from other users (e.g. CF prices)

	UsageCount int       `json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AutoReplyLog records every comment the engine acted on (or failed to),
// so the owner can audit what the bot did.
type AutoReplyLog struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	RuleID   uint   `json:"rule_id"`
	RuleName string `json:"rule_name"` // snapshot — survives rule edits/deletes
	Platform string `json:"platform"`
	// CommentID is the platform comment id; FromName/FromID identify the
	// commenter; CommentText is what they wrote.
	CommentID   string `json:"comment_id"`
	FromID      string `json:"from_id"`
	FromName    string `json:"from_name"`
	CommentText string `json:"comment_text"`
	// Actions is a comma list of what ran: reply, private_reply, hide.
	Actions string `json:"actions"`
	// Status: success | partial | failed. Error keeps the first failure.
	Status    string    `json:"status"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
}
