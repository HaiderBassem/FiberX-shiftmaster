package models

import "time"

type IPBlock struct {
	IPAddress string    `json:"ip_address"`
	Reason    string    `json:"reason"`
	BlockedAt time.Time `json:"blocked_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
