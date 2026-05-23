package contract

import "time"

const SearchEventSchemaVersion = 1

type SearchEvent struct {
	SchemaVersion int       `json:"schema_version"`
	EventID       string    `json:"event_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Query         string    `json:"query"`

	UserIDHash    string `json:"user_id_hash,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	DeviceIDHash  string `json:"device_id_hash,omitempty"`
	IPHash        string `json:"ip_hash,omitempty"`
	UserAgentHash string `json:"user_agent_hash,omitempty"`

	Region   string `json:"region,omitempty"`
	Locale   string `json:"locale,omitempty"`
	Platform string `json:"platform,omitempty"`

	IsBot bool `json:"is_bot,omitempty"`
}

func (e SearchEvent) ActorKey() string {
	switch {
	case e.UserIDHash != "":
		return e.UserIDHash
	case e.DeviceIDHash != "":
		return e.DeviceIDHash
	case e.IPHash != "":
		return e.IPHash
	case e.SessionID != "":
		return e.SessionID
	default:
		return ""
	}
}
