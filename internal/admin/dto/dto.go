package dto

type AuditLogResponse struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  *string        `json:"user_agent,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  string         `json:"created_at"`
}

type ListAuditLogsFilter struct {
	ActorID    *string `json:"actor_id,omitempty"`
	Action     *string `json:"action,omitempty"`
	TargetType *string `json:"target_type,omitempty"`
	From       *string `json:"from,omitempty"`
	To         *string `json:"to,omitempty"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
}
