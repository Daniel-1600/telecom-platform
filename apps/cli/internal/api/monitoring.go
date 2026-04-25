package api

import "time"

// Alert represents a monitoring alert
type Alert struct {
	ID       string    `json:"id"`
	Severity string    `json:"severity"`
	Service  string    `json:"service"`
	Message  string    `json:"message"`
	Time     time.Time `json:"time"`
}

// ListAlerts returns alerts
func (c *Client) ListAlerts() ([]Alert, error) {
	var resp struct {
		Data []Alert `json:"data"`
	}
	if err := c.doGetJSON("/api/v1/alerts", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// SystemStats represents system health/metrics
type SystemStats struct {
	ActiveSessions   int     `json:"active_sessions"`
	TotalAccounts    int     `json:"total_accounts"`
	BlockedUsers     int     `json:"blocked_users"`
	LowBalanceAlerts int     `json:"low_balance_alerts"`
	CPUUsage         float64 `json:"cpu_usage"`
	MemoryUsage      float64 `json:"memory_usage"`
	Uptime           string  `json:"uptime"`
}

// GetSystemStats retrieves system stats
func (c *Client) GetSystemStats() (*SystemStats, error) {
	var stats SystemStats
	if err := c.doGetJSON("/api/v1/system/stats", &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// HealthStatus represents a health check result
type HealthStatus struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	LastCheck time.Time `json:"last_check"`
}

// GetHealth returns per-service health
func (c *Client) GetHealth() ([]HealthStatus, error) {
	var resp struct {
		Data []HealthStatus `json:"data"`
	}
	if err := c.doGetJSON("/api/v1/health", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
