package tenant

import (
	"time"
)

// TenantFilter defines filtering options for tenant queries
type TenantFilter struct {
	ID        string       `json:"id,omitempty"`
	Name      string       `json:"name,omitempty"`
	Domain    string       `json:"domain,omitempty"`
	Status    TenantStatus `json:"status,omitempty"`
	Plan      TenantPlan   `json:"plan,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
	SortBy    string       `json:"sort_by,omitempty"`
	SortOrder string       `json:"sort_order,omitempty"`
}

// CreateTenantRequest represents a request to create a new tenant
type CreateTenantRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Domain      string                 `json:"domain" binding:"required"`
	Plan        TenantPlan             `json:"plan" binding:"required"`
	MaxUsers    int                    `json:"max_users"`
	MaxProfiles int                    `json:"max_profiles"`
	MaxCarriers int                    `json:"max_carriers"`
	Settings    *TenantSettings        `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTenantRequest represents a request to update a tenant
type UpdateTenantRequest struct {
	Name        *string                `json:"name,omitempty"`
	Status      *TenantStatus          `json:"status,omitempty"`
	Plan        *TenantPlan            `json:"plan,omitempty"`
	MaxUsers    *int                   `json:"max_users,omitempty"`
	MaxProfiles *int                   `json:"max_profiles,omitempty"`
	MaxCarriers *int                   `json:"max_carriers,omitempty"`
	Settings    *TenantSettings        `json:"settings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TenantUserFilter defines filtering options for tenant user queries
type TenantUserFilter struct {
	TenantID string           `json:"tenant_id,omitempty"`
	UserID   string           `json:"user_id,omitempty"`
	Email    string           `json:"email,omitempty"`
	Role     TenantRole       `json:"role,omitempty"`
	Status   TenantUserStatus `json:"status,omitempty"`
	Limit    int              `json:"limit,omitempty"`
	Offset   int              `json:"offset,omitempty"`
}

// CreateTenantUserRequest represents a request to add a user to a tenant
type CreateTenantUserRequest struct {
	TenantID string     `json:"tenant_id" binding:"required"`
	UserID   string     `json:"user_id" binding:"required"`
	Email    string     `json:"email" binding:"required,email"`
	Role     TenantRole `json:"role" binding:"required"`
}

// UpdateTenantUserRequest represents a request to update a tenant user
type UpdateTenantUserRequest struct {
	Role   *TenantRole       `json:"role,omitempty"`
	Status *TenantUserStatus `json:"status,omitempty"`
}

// CreateAPIKeyRequest represents a request to create a new API key
type CreateAPIKeyRequest struct {
	Name        string     `json:"name" binding:"required"`
	Permissions []string   `json:"permissions"`
	RateLimit   int        `json:"rate_limit"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name        *string       `json:"name,omitempty"`
	Permissions []string      `json:"permissions,omitempty"`
	RateLimit   *int          `json:"rate_limit,omitempty"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
	Status      *APIKeyStatus `json:"status,omitempty"`
}

// TenantUsageFilter defines filtering options for tenant usage queries
type TenantUsageFilter struct {
	TenantID     string    `json:"tenant_id,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	PeriodStart  time.Time `json:"period_start,omitempty"`
	PeriodEnd    time.Time `json:"period_end,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Offset       int       `json:"offset,omitempty"`
}

// TenantUsageStats represents usage statistics for a tenant
type TenantUsageStats struct {
	TenantID          string                 `json:"tenant_id"`
	TotalUsers        int                    `json:"total_users"`
	ActiveUsers       int                    `json:"active_users"`
	TotalProfiles     int                    `json:"total_profiles"`
	ActiveProfiles    int                    `json:"active_profiles"`
	TotalCarriers     int                    `json:"total_carriers"`
	ActiveCarriers    int                    `json:"active_carriers"`
	APIRequests       int64                  `json:"api_requests"`
	StorageUsed       int64                  `json:"storage_used"`
	LastActivity      time.Time              `json:"last_activity"`
	ResourceBreakdown map[string]int64       `json:"resource_breakdown"`
	QuotaStatus       map[string]QuotaStatus `json:"quota_status"`
}

// QuotaStatus represents the status of a resource quota
type QuotaStatus struct {
	Used      int     `json:"used"`
	Limit     int     `json:"limit"`
	Remaining int     `json:"remaining"`
	Percent   float64 `json:"percent"`
	Warning   bool    `json:"warning"`
	Critical  bool    `json:"critical"`
}

// TenantContext represents tenant context for request processing
type TenantContext struct {
	TenantID   string                 `json:"tenant_id"`
	TenantName string                 `json:"tenant_name"`
	Plan       TenantPlan             `json:"plan"`
	UserID     string                 `json:"user_id"`
	UserRole   TenantRole             `json:"user_role"`
	Settings   *TenantSettings        `json:"settings"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ResourceQuota represents resource quota configuration
type ResourceQuota struct {
	ResourceType     string  `json:"resource_type"`
	Limit            int     `json:"limit"`
	Period           string  `json:"period"` // daily, monthly, yearly
	SoftLimit        bool    `json:"soft_limit"`
	WarningThreshold float64 `json:"warning_threshold"` // percentage
}

// ResourceUsage represents actual resource usage
type ResourceUsage struct {
	ResourceType string    `json:"resource_type"`
	Count        int       `json:"count"`
	LastUpdated  time.Time `json:"last_updated"`
}

// TenantConfig represents tenant-specific configuration
type TenantConfig struct {
	TenantID string                 `json:"tenant_id"`
	Config   map[string]interface{} `json:"config"`
	Settings *TenantSettings        `json:"settings"`
	Quotas   []ResourceQuota        `json:"quotas"`
	Features map[string]bool        `json:"features"`
}

// TenantEvent represents events related to a tenant
type TenantEvent struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	UserID    string                 `json:"user_id"`
	EventType TenantEventType        `json:"event_type"`
	EventData map[string]interface{} `json:"event_data"`
	Timestamp time.Time              `json:"timestamp"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
}

// TenantEventType represents types of tenant events
type TenantEventType string

const (
	TenantEventCreated       TenantEventType = "tenant_created"
	TenantEventUpdated       TenantEventType = "tenant_updated"
	TenantEventDeleted       TenantEventType = "tenant_deleted"
	TenantEventUserAdded     TenantEventType = "user_added"
	TenantEventUserRemoved   TenantEventType = "user_removed"
	TenantEventUserUpdated   TenantEventType = "user_updated"
	TenantEventAPIKeyCreated TenantEventType = "api_key_created"
	TenantEventAPIKeyRevoked TenantEventType = "api_key_revoked"
	TenantEventQuotaExceeded TenantEventType = "quota_exceeded"
	TenantEventQuotaWarning  TenantEventType = "quota_warning"
	TenantEventLogin         TenantEventType = "login"
	TenantEventLogout        TenantEventType = "logout"
)

// TenantMetrics represents metrics for monitoring tenant health
type TenantMetrics struct {
	TenantID      string    `json:"tenant_id"`
	ActiveUsers   int       `json:"active_users"`
	TotalRequests int64     `json:"total_requests"`
	ErrorRate     float64   `json:"error_rate"`
	ResponseTime  float64   `json:"response_time"`
	StorageUsed   int64     `json:"storage_used"`
	LastActivity  time.Time `json:"last_activity"`
	HealthScore   float64   `json:"health_score"`
	Alerts        []string  `json:"alerts"`
}
