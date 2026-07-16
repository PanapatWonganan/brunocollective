package models

import "time"

type Customer struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Name    string `json:"name" gorm:"not null"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Notes   string `json:"notes"`
	// Membership: members get a flat discount on every order (see
	// handlers/member.go). A customer becomes a member by registering on the
	// storefront, by admin toggle, or automatically once they have a prior
	// order (auto-membership for returning customers).
	IsMember    bool       `json:"is_member"`
	MemberSince *time.Time `json:"member_since"`
	// PasswordHash is set when the customer registers a storefront member
	// account (login by phone). Empty = no account yet. Never serialized.
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
