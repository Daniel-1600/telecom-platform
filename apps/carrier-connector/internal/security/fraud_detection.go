package security

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// FraudType represents different types of fraud
type FraudType string

const (
	FraudTypeAccountTakeover   FraudType = "account_takeover"
	FraudTypeSubscriptionFraud FraudType = "subscription_fraud"
	FraudTypePaymentFraud      FraudType = "payment_fraud"
	FraudTypeUsageAnomaly      FraudType = "usage_anomaly"
	FraudTypeIdentityFraud     FraudType = "identity_fraud"
	FraudTypeSIMSwap           FraudType = "sim_swap"
)

// FraudSeverity represents the severity of fraud detection
type FraudSeverity string

const (
	FraudSeverityLow      FraudSeverity = "low"
	FraudSeverityMedium   FraudSeverity = "medium"
	FraudSeverityHigh     FraudSeverity = "high"
	FraudSeverityCritical FraudSeverity = "critical"
)

// FraudAlert represents a fraud detection alert
type FraudAlert struct {
	ID          string         `json:"id"`
	Type        FraudType      `json:"type"`
	Severity    FraudSeverity  `json:"severity"`
	ProfileID   string         `json:"profile_id"`
	Description string         `json:"description"`
	RiskScore   float64        `json:"risk_score"` // 0-100
	Evidence    []string       `json:"evidence"`
	IPAddress   string         `json:"ip_address"`
	UserAgent   string         `json:"user_agent"`
	Location    string         `json:"location"`
	Timestamp   time.Time      `json:"timestamp"`
	Status      string         `json:"status"` // "new", "investigating", "resolved", "false_positive"
	Actions     []string       `json:"actions_taken"`
	Metadata    map[string]any `json:"metadata"`
}

// FraudPattern represents a fraud detection pattern
type FraudPattern struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        FraudType `json:"type"`
	Description string    `json:"description"`
	Threshold   float64   `json:"threshold"`
	Weight      float64   `json:"weight"`
	Enabled     bool      `json:"enabled"`
}

// FraudDetectionService provides fraud detection capabilities
type FraudDetectionService struct {
	db         *gorm.DB
	logger     *logrus.Logger
	patterns   []FraudPattern
	alerts     []*FraudAlert
	mu         sync.RWMutex
	riskModels map[string]*RiskModel
}

// RiskModel represents a machine learning model for fraud detection
type RiskModel struct {
	Name        string    `json:"name"`
	Type        FraudType `json:"type"`
	Version     string    `json:"version"`
	LastTrained time.Time `json:"last_trained"`
	Accuracy    float64   `json:"accuracy"`
	Threshold   float64   `json:"threshold"`
}

// FraudDetectionConfig configures the fraud detection service
type FraudDetectionConfig struct {
	EnableMLModels     bool
	RiskThreshold      float64
	AlertRetention     int // days
	AutoBlockThreshold float64
}

// NewFraudDetectionService creates a new fraud detection service
func NewFraudDetectionService(db *gorm.DB, logger *logrus.Logger, config FraudDetectionConfig) *FraudDetectionService {
	service := &FraudDetectionService{
		db:         db,
		logger:     logger,
		patterns:   getDefaultFraudPatterns(),
		alerts:     make([]*FraudAlert, 0),
		riskModels: make(map[string]*RiskModel),
	}

	// Initialize ML models if enabled
	if config.EnableMLModels {
		service.initializeRiskModels()
	}

	go service.cleanupOldAlerts(config.AlertRetention)

	return service
}

// AnalyzeTransaction analyzes a transaction for fraud
func (s *FraudDetectionService) AnalyzeTransaction(ctx context.Context, transaction map[string]interface{}) (*FraudAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert := &FraudAlert{
		ID:        fmt.Sprintf("fraud-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Status:    "new",
		Metadata:  make(map[string]any),
	}

	// Extract transaction details
	if profileID, ok := transaction["profile_id"].(string); ok {
		alert.ProfileID = profileID
	}
	if ip, ok := transaction["ip_address"].(string); ok {
		alert.IPAddress = ip
	}
	if ua, ok := transaction["user_agent"].(string); ok {
		alert.UserAgent = ua
	}

	// Run fraud detection patterns
	riskScore := 0.0
	evidence := make([]string, 0)

	for _, pattern := range s.patterns {
		if !pattern.Enabled {
			continue
		}

		patternScore, patternEvidence := s.evaluatePattern(ctx, transaction, pattern)
		if patternScore > 0 {
			riskScore += patternScore * pattern.Weight
			evidence = append(evidence, patternEvidence...)
		}
	}

	// Apply ML models if available
	if mlScore := s.applyRiskModels(ctx, transaction); mlScore > 0 {
		riskScore = (riskScore + mlScore) / 2
		evidence = append(evidence, "ML model anomaly detected")
	}

	alert.RiskScore = math.Min(100, riskScore)
	alert.Evidence = evidence

	// Determine severity and type
	alert.Severity = s.determineSeverity(alert.RiskScore)
	alert.Type = s.determineFraudType(evidence)
	alert.Description = s.generateDescription(alert.Type, alert.Severity, evidence)

	// Take automated actions if needed
	if alert.RiskScore >= 80 {
		alert.Actions = append(alert.Actions, "auto_blocked")
		s.blockProfile(ctx, alert.ProfileID)
	} else if alert.RiskScore >= 60 {
		alert.Actions = append(alert.Actions, "flagged_for_review")
	}

	s.alerts = append(s.alerts, alert)

	s.logger.WithFields(logrus.Fields{
		"alert_id":   alert.ID,
		"risk_score": alert.RiskScore,
		"type":       alert.Type,
		"severity":   alert.Severity,
	}).Warn("Fraud detected")

	return alert, nil
}

// GetFraudAlerts retrieves fraud alerts
func (s *FraudDetectionService) GetFraudAlerts(ctx context.Context, filter FraudAlertFilter) ([]*FraudAlert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*FraudAlert, 0)

	for _, alert := range s.alerts {
		if s.matchesFilter(alert, filter) {
			filtered = append(filtered, alert)
		}
	}

	return filtered, nil
}

// UpdateAlertStatus updates the status of a fraud alert
func (s *FraudDetectionService) UpdateAlertStatus(ctx context.Context, alertID, status string, actions []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, alert := range s.alerts {
		if alert.ID == alertID {
			alert.Status = status
			if len(actions) > 0 {
				alert.Actions = append(alert.Actions, actions...)
			}
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// GetFraudMetrics returns fraud detection metrics
func (s *FraudDetectionService) GetFraudMetrics(ctx context.Context, period string) (*FraudMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := &FraudMetrics{
		Period:      period,
		GeneratedAt: time.Now(),
		ByType:      make(map[FraudType]int64),
		BySeverity:  make(map[FraudSeverity]int64),
	}

	startDate, endDate := s.getPeriodDates(period)

	for _, alert := range s.alerts {
		if alert.Timestamp.After(startDate) && alert.Timestamp.Before(endDate) {
			metrics.TotalAlerts++
			metrics.ByType[alert.Type]++
			metrics.BySeverity[alert.Severity]++

			if alert.Status == "resolved" {
				metrics.ResolvedAlerts++
			} else if alert.Status == "false_positive" {
				metrics.FalsePositives++
			}
		}
	}

	// Calculate rates
	if metrics.TotalAlerts > 0 {
		metrics.ResolutionRate = float64(metrics.ResolvedAlerts) / float64(metrics.TotalAlerts) * 100
		metrics.FalsePositiveRate = float64(metrics.FalsePositives) / float64(metrics.TotalAlerts) * 100
	}

	return metrics, nil
}

// FraudMetrics represents fraud detection metrics
type FraudMetrics struct {
	Period            string                  `json:"period"`
	TotalAlerts       int64                   `json:"total_alerts"`
	ResolvedAlerts    int64                   `json:"resolved_alerts"`
	FalsePositives    int64                   `json:"false_positives"`
	ResolutionRate    float64                 `json:"resolution_rate_pct"`
	FalsePositiveRate float64                 `json:"false_positive_rate_pct"`
	ByType            map[FraudType]int64     `json:"by_type"`
	BySeverity        map[FraudSeverity]int64 `json:"by_severity"`
	GeneratedAt       time.Time               `json:"generated_at"`
}

// FraudAlertFilter filters fraud alerts
type FraudAlertFilter struct {
	Type     FraudType     `json:"type,omitempty"`
	Severity FraudSeverity `json:"severity,omitempty"`
	Status   string        `json:"status,omitempty"`
	FromDate *time.Time    `json:"from_date,omitempty"`
	ToDate   *time.Time    `json:"to_date,omitempty"`
	Limit    int           `json:"limit,omitempty"`
}

// getDefaultFraudPatterns returns default fraud detection patterns
func getDefaultFraudPatterns() []FraudPattern {
	return []FraudPattern{
		{
			ID:          "multiple_subscriptions",
			Name:        "Multiple Subscriptions",
			Type:        FraudTypeSubscriptionFraud,
			Description: "Multiple active subscriptions from same profile",
			Threshold:   3,
			Weight:      0.3,
			Enabled:     true,
		},
		{
			ID:          "rapid_subscription",
			Name:        "Rapid Subscription Creation",
			Type:        FraudTypeSubscriptionFraud,
			Description: "Multiple subscriptions created in short time",
			Threshold:   5,
			Weight:      0.4,
			Enabled:     true,
		},
		{
			ID:          "unusual_location",
			Name:        "Unusual Location Access",
			Type:        FraudTypeAccountTakeover,
			Description: "Access from unusual geographic location",
			Threshold:   0.8,
			Weight:      0.5,
			Enabled:     true,
		},
		{
			ID:          "high_usage_spike",
			Name:        "High Usage Spike",
			Type:        FraudTypeUsageAnomaly,
			Description: "Sudden spike in data usage",
			Threshold:   1000, // MB
			Weight:      0.3,
			Enabled:     true,
		},
		{
			ID:          "payment_failure_pattern",
			Name:        "Payment Failure Pattern",
			Type:        FraudTypePaymentFraud,
			Description: "Multiple payment failures",
			Threshold:   3,
			Weight:      0.6,
			Enabled:     true,
		},
		{
			ID:          "sim_swap_indication",
			Name:        "SIM Swap Indication",
			Type:        FraudTypeSIMSwap,
			Description: "Indicators of SIM swap attack",
			Threshold:   0.7,
			Weight:      0.8,
			Enabled:     true,
		},
	}
}

// evaluatePattern evaluates a specific fraud pattern
func (s *FraudDetectionService) evaluatePattern(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	evidence := make([]string, 0)
	score := 0.0

	switch pattern.ID {
	case "multiple_subscriptions":
		score, evidence = s.checkMultipleSubscriptions(ctx, transaction, pattern)
	case "rapid_subscription":
		score, evidence = s.checkRapidSubscription(ctx, transaction, pattern)
	case "unusual_location":
		score, evidence = s.checkUnusualLocation(ctx, transaction, pattern)
	case "high_usage_spike":
		score, evidence = s.checkUsageSpike(ctx, transaction, pattern)
	case "payment_failure_pattern":
		score, evidence = s.checkPaymentFailures(ctx, transaction, pattern)
	case "sim_swap_indication":
		score, evidence = s.checkSIMSwap(ctx, transaction, pattern)
	}

	return score, evidence
}

// checkMultipleSubscriptions checks for multiple subscriptions
func (s *FraudDetectionService) checkMultipleSubscriptions(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0, nil
	}

	var count int64
	s.db.WithContext(ctx).Table("rate_plan_subscriptions").
		Where("profile_id = ? AND status = ?", profileID, "active").
		Count(&count)

	if count > int64(pattern.Threshold) {
		return float64(count) * 20, []string{fmt.Sprintf("Found %d active subscriptions", count)}
	}

	return 0, nil
}

// checkRapidSubscription checks for rapid subscription creation
func (s *FraudDetectionService) checkRapidSubscription(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0, nil
	}

	var count int64
	s.db.WithContext(ctx).Table("rate_plan_subscriptions").
		Where("profile_id = ? AND created_at > ?", profileID, time.Now().Add(-time.Hour)).
		Count(&count)

	if count > int64(pattern.Threshold) {
		return float64(count) * 15, []string{fmt.Sprintf("Created %d subscriptions in last hour", count)}
	}

	return 0, nil
}

// checkUnusualLocation checks for unusual geographic access
func (s *FraudDetectionService) checkUnusualLocation(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	ipAddress, ok := transaction["ip_address"].(string)
	if !ok {
		return 0, nil
	}

	// Simplified location check - in production use GeoIP
	country := s.getCountryFromIP(ipAddress)

	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0, nil
	}

	// Check if this is a new country for this profile
	var prevCountry string
	s.db.WithContext(ctx).Table("profiles").
		Where("id = ?", profileID).
		Select("country").
		Scan(&prevCountry)

	if prevCountry != "" && country != prevCountry {
		return 70, []string{fmt.Sprintf("Access from new country: %s (previous: %s)", country, prevCountry)}
	}

	return 0, nil
}

// checkUsageSpike checks for unusual usage patterns
func (s *FraudDetectionService) checkUsageSpike(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0, nil
	}

	// Get current usage
	var currentUsage int64
	s.db.WithContext(ctx).Table("rate_plan_usage").
		Where("profile_id = ? AND created_at > ?", profileID, time.Now().Add(-24*time.Hour)).
		Select("COALESCE(SUM(data_used), 0)").
		Scan(&currentUsage)

	// Get average usage for comparison
	var avgUsage int64
	s.db.WithContext(ctx).Table("rate_plan_usage").
		Where("profile_id = ? AND created_at BETWEEN ? AND ?",
			profileID, time.Now().Add(-30*24*time.Hour), time.Now().Add(-24*time.Hour)).
		Select("COALESCE(AVG(data_used), 0)").
		Scan(&avgUsage)

	if avgUsage > 0 && currentUsage > avgUsage*5 && currentUsage > int64(pattern.Threshold) {
		return 60, []string{fmt.Sprintf("Usage spike: %d MB (avg: %d MB)", currentUsage, avgUsage)}
	}

	return 0, nil
}

// checkPaymentFailures checks for payment failure patterns
func (s *FraudDetectionService) checkPaymentFailures(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0, nil
	}

	var count int64
	s.db.WithContext(ctx).Table("billing_transactions").
		Where("profile_id = ? AND status = ? AND created_at > ?", profileID, "failed", time.Now().Add(-24*time.Hour)).
		Count(&count)

	if count > int64(pattern.Threshold) {
		return float64(count) * 25, []string{fmt.Sprintf("%d payment failures in last 24 hours", count)}
	}

	return 0, nil
}

// checkSIMSwap checks for SIM swap indicators
func (s *FraudDetectionService) checkSIMSwap(ctx context.Context, transaction map[string]interface{}, pattern FraudPattern) (float64, []string) {
	// Check for multiple profile changes in short time
	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0, nil
	}

	var updates int64
	s.db.WithContext(ctx).Table("profiles").
		Where("id = ? AND updated_at > ?", profileID, time.Now().Add(-time.Hour)).
		Count(&updates)

	if updates > 2 {
		return 80, []string{fmt.Sprintf("%d profile updates in last hour (possible SIM swap)", updates)}
	}

	return 0, nil
}

// getCountryFromIP simulates GeoIP lookup
func (s *FraudDetectionService) getCountryFromIP(ip string) string {
	// Simplified - in production use actual GeoIP database
	if ip == "127.0.0.1" || ip == "::1" {
		return "US"
	}
	return "Unknown"
}

// determineSeverity determines fraud severity from risk score
func (s *FraudDetectionService) determineSeverity(score float64) FraudSeverity {
	switch {
	case score >= 80:
		return FraudSeverityCritical
	case score >= 60:
		return FraudSeverityHigh
	case score >= 40:
		return FraudSeverityMedium
	default:
		return FraudSeverityLow
	}
}

// determineFraudType determines fraud type from evidence
func (s *FraudDetectionService) determineFraudType(evidence []string) FraudType {
	for _, ev := range evidence {
		if containsIgnoreCaseFraud(ev, "subscription") {
			return FraudTypeSubscriptionFraud
		}
		if containsIgnoreCaseFraud(ev, "payment") {
			return FraudTypePaymentFraud
		}
		if containsIgnoreCaseFraud(ev, "location") || containsIgnoreCaseFraud(ev, "country") {
			return FraudTypeAccountTakeover
		}
		if containsIgnoreCaseFraud(ev, "usage") {
			return FraudTypeUsageAnomaly
		}
		if containsIgnoreCaseFraud(ev, "sim swap") {
			return FraudTypeSIMSwap
		}
	}
	return FraudTypeSubscriptionFraud // Default
}

// ...
// generateDescription generates alert description
func (s *FraudDetectionService) generateDescription(fraudType FraudType, severity FraudSeverity, evidence []string) string {
	desc := fmt.Sprintf("%s %s fraud detected", string(severity), string(fraudType))
	if len(evidence) > 0 {
		desc += fmt.Sprintf(": %s", evidence[0])
	}
	return desc
}

// blockProfile blocks a profile for fraud
func (s *FraudDetectionService) blockProfile(ctx context.Context, profileID string) {
	s.db.WithContext(ctx).Table("profiles").
		Where("id = ?", profileID).
		Updates(map[string]interface{}{
			"status":       "blocked",
			"blocked_at":   time.Now(),
			"block_reason": "fraud_detection",
		})

	s.logger.WithField("profile_id", profileID).Warn("Profile blocked due to fraud detection")
}

// applyRiskModels applies ML models for fraud detection
func (s *FraudDetectionService) applyRiskModels(ctx context.Context, transaction map[string]interface{}) float64 {
	// Simplified ML model application
	// In production, this would use actual trained models
	profileID, ok := transaction["profile_id"].(string)
	if !ok {
		return 0
	}

	// Get profile history
	var subscriptionCount int64
	s.db.WithContext(ctx).Table("rate_plan_subscriptions").
		Where("profile_id = ?", profileID).
		Count(&subscriptionCount)

	var paymentFailures int64
	s.db.WithContext(ctx).Table("billing_transactions").
		Where("profile_id = ? AND status = ?", profileID, "failed").
		Count(&paymentFailures)

	// Simple risk scoring
	risk := 0.0
	if subscriptionCount > 5 {
		risk += 30
	}
	if paymentFailures > 2 {
		risk += 40
	}

	return risk
}

// initializeRiskModels initializes ML risk models
func (s *FraudDetectionService) initializeRiskModels() {
	s.riskModels["subscription_fraud"] = &RiskModel{
		Name:        "Subscription Fraud Model",
		Type:        FraudTypeSubscriptionFraud,
		Version:     "1.0",
		LastTrained: time.Now().AddDate(0, -1, 0),
		Accuracy:    0.92,
		Threshold:   0.75,
	}

	s.riskModels["account_takeover"] = &RiskModel{
		Name:        "Account Takeover Model",
		Type:        FraudTypeAccountTakeover,
		Version:     "1.0",
		LastTrained: time.Now().AddDate(0, -1, 0),
		Accuracy:    0.88,
		Threshold:   0.80,
	}
}

// matchesFilter checks if alert matches filter criteria
func (s *FraudDetectionService) matchesFilter(alert *FraudAlert, filter FraudAlertFilter) bool {
	if filter.Type != "" && alert.Type != filter.Type {
		return false
	}
	if filter.Severity != "" && alert.Severity != filter.Severity {
		return false
	}
	if filter.Status != "" && alert.Status != filter.Status {
		return false
	}
	if filter.FromDate != nil && alert.Timestamp.Before(*filter.FromDate) {
		return false
	}
	if filter.ToDate != nil && alert.Timestamp.After(*filter.ToDate) {
		return false
	}
	return true
}

// getPeriodDates returns start and end dates for a period
func (s *FraudDetectionService) getPeriodDates(period string) (time.Time, time.Time) {
	now := time.Now()

	switch period {
	case "daily":
		return now.Truncate(24 * time.Hour), now
	case "weekly":
		return now.AddDate(0, 0, -7), now
	case "monthly":
		return now.AddDate(0, -1, 0), now
	case "quarterly":
		return now.AddDate(0, -3, 0), now
	default:
		return now.AddDate(0, -1, 0), now
	}
}

// cleanupOldAlerts removes old alerts
func (s *FraudDetectionService) cleanupOldAlerts(retentionDays int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().AddDate(0, 0, -retentionDays)

		filtered := make([]*FraudAlert, 0)
		for _, alert := range s.alerts {
			if alert.Timestamp.After(cutoff) {
				filtered = append(filtered, alert)
			}
		}
		s.alerts = filtered
		s.mu.Unlock()
	}
}

// containsIgnoreCaseFraud checks if string contains substring (case insensitive)
func containsIgnoreCaseFraud(s, substr string) bool {
	// Simplified case-insensitive check
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
