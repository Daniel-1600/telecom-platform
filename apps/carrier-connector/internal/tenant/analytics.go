package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// TenantAnalyticsService provides analytics and monitoring for tenants
type TenantAnalyticsService struct {
	repository       Repository
	metricsCollector MetricsCollector
	eventPublisher   EventPublisher
	logger           *logrus.Logger
}

// NewTenantAnalyticsService creates a new tenant analytics service
func NewTenantAnalyticsService(
	repository Repository,
	metricsCollector MetricsCollector,
	eventPublisher EventPublisher,
	logger *logrus.Logger,
) *TenantAnalyticsService {
	return &TenantAnalyticsService{
		repository:       repository,
		metricsCollector: metricsCollector,
		eventPublisher:   eventPublisher,
		logger:           logger,
	}
}

// GetTenantDashboard returns dashboard data for a tenant
func (s *TenantAnalyticsService) GetTenantDashboard(ctx context.Context, tenantID string) (*TenantDashboard, error) {
	// Get usage stats
	usageStats, err := s.repository.GetUsageStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Get tenant metrics
	metrics, err := s.GetTenantMetrics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant metrics: %w", err)
	}

	// Get recent events
	events, err := s.repository.ListEvents(ctx, tenantID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant events: %w", err)
	}

	// Get quota status
	var quotaStatus []*TenantUsage
	quotaFilter := &TenantUsageFilter{TenantID: tenantID}
	quotaStatus, err = s.repository.ListUsage(ctx, quotaFilter)
	if err != nil {
		// Handle gracefully - quota might not exist yet
		quotaStatus = []*TenantUsage{}
	}

	dashboard := &TenantDashboard{
		TenantID:     tenantID,
		UsageStats:   usageStats,
		Metrics:      metrics,
		RecentEvents: events,
		QuotaStatus:  quotaStatus,
		LastUpdated:  time.Now(),
	}

	return dashboard, nil
}

// GetTenantMetrics returns comprehensive metrics for a tenant
func (s *TenantAnalyticsService) GetTenantMetrics(ctx context.Context, tenantID string) (*TenantMetrics, error) {
	// Get basic metrics from repository
	basicMetrics, err := s.repository.GetUsageStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Get events for activity analysis
	events, err := s.repository.ListEvents(ctx, tenantID, 1000)
	if err != nil {
		return nil, err
	}

	// Build comprehensive metrics
	metrics := &TenantMetrics{
		TenantID:      tenantID,
		ActiveUsers:   basicMetrics.ActiveUsers,
		TotalRequests: 0,
		ErrorRate:     0,
		ResponseTime:  0,
		StorageUsed:   0,
		LastActivity:  time.Time{},
		HealthScore:   100.0,
		Alerts:        []string{},
	}

	// Analyze events for metrics
	errorCount := 0
	totalRequests := 0
	var totalResponseTime time.Duration
	lastActivity := time.Time{}

	for _, event := range events {
		if event.Timestamp.After(lastActivity) {
			lastActivity = event.Timestamp
		}

		switch event.EventType {
		case "api_request":
			totalRequests++
			if statusCode, exists := event.EventData["status_code"]; exists {
				if code, ok := statusCode.(float64); ok && code >= 400 {
					errorCount++
				}
			}
			if responseTime, exists := event.EventData["response_time"]; exists {
				if rt, ok := responseTime.(float64); ok {
					totalResponseTime += time.Duration(rt) * time.Millisecond
				}
			}
		case "resource_created", "resource_updated", "resource_deleted":
			metrics.TotalRequests++
		}
	}

	if totalRequests > 0 {
		metrics.ErrorRate = float64(errorCount) / float64(totalRequests) * 100
		metrics.ResponseTime = float64(totalResponseTime) / float64(totalRequests) / float64(time.Millisecond)
	}

	metrics.TotalRequests = int64(totalRequests)
	metrics.LastActivity = lastActivity

	// Calculate health score and alerts
	metrics.HealthScore = s.calculateHealthScore(basicMetrics, metrics.ErrorRate)
	metrics.Alerts = s.generateAlerts(basicMetrics, metrics.ErrorRate)

	return metrics, nil
}

// GetUsageAnalytics returns detailed usage analytics for a tenant
func (s *TenantAnalyticsService) GetUsageAnalytics(ctx context.Context, tenantID string, timeRange string) (*TenantUsageAnalytics, error) {
	// Parse time range
	startDate, endDate := s.parseTimeRange(timeRange)

	// Get usage records
	usageFilter := &TenantUsageFilter{
		TenantID:    tenantID,
		PeriodStart: startDate,
		PeriodEnd:   endDate,
	}

	usageRecords, err := s.repository.ListUsage(ctx, usageFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage records: %w", err)
	}

	// Build analytics
	analytics := &TenantUsageAnalytics{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		StartDate:   startDate,
		EndDate:     endDate,
		UsageByType: make(map[string]*ResourceUsageAnalytics),
		Trends:      make(map[string][]*UsageTrend),
		Peaks:       make(map[string]*UsagePeak),
	}

	// Process usage records
	for _, usage := range usageRecords {
		if _, exists := analytics.UsageByType[usage.ResourceType]; !exists {
			analytics.UsageByType[usage.ResourceType] = &ResourceUsageAnalytics{
				ResourceType: usage.ResourceType,
				TotalUsage:   0,
				AverageUsage: 0,
				PeakUsage:    0,
				PeakTime:     time.Time{},
			}
		}

		resourceAnalytics := analytics.UsageByType[usage.ResourceType]
		resourceAnalytics.TotalUsage += usage.QuotaUsed

		if usage.QuotaUsed > resourceAnalytics.PeakUsage {
			resourceAnalytics.PeakUsage = usage.QuotaUsed
			resourceAnalytics.PeakTime = usage.UpdatedAt
		}
	}

	// Calculate averages
	for _, resourceAnalytics := range analytics.UsageByType {
		if len(usageRecords) > 0 {
			resourceAnalytics.AverageUsage = resourceAnalytics.TotalUsage / len(usageRecords)
		}
	}

	return analytics, nil
}

// GetPerformanceAnalytics returns performance analytics for a tenant
func (s *TenantAnalyticsService) GetPerformanceAnalytics(ctx context.Context, tenantID string, timeRange string) (*TenantPerformanceAnalytics, error) {
	// Get events for performance analysis
	events, err := s.repository.ListEvents(ctx, tenantID, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant events: %w", err)
	}

	// Parse time range
	startDate, endDate := s.parseTimeRange(timeRange)

	// Build performance analytics
	analytics := &TenantPerformanceAnalytics{
		TenantID:            tenantID,
		TimeRange:           timeRange,
		StartDate:           startDate,
		EndDate:             endDate,
		APIPerformance:      &APIPerformance{},
		ResourcePerformance: make(map[string]*ResourcePerformance),
		Errors:              []*ErrorEvent{},
		SlowQueries:         []*SlowQuery{},
	}

	// Process events
	apiRequests := []*APIRequestEvent{}
	for _, event := range events {
		if event.Timestamp.Before(startDate) || event.Timestamp.After(endDate) {
			continue
		}

		switch event.EventType {
		case "api_request":
			apiRequest := s.parseAPIRequestEvent(event)
			if apiRequest != nil {
				apiRequests = append(apiRequests, apiRequest)
			}
		case "error":
			errorEvent := s.parseErrorEvent(event)
			if errorEvent != nil {
				analytics.Errors = append(analytics.Errors, errorEvent)
			}
		case "slow_query":
			slowQuery := s.parseSlowQueryEvent(event)
			if slowQuery != nil {
				analytics.SlowQueries = append(analytics.SlowQueries, slowQuery)
			}
		}
	}

	// Calculate API performance metrics
	analytics.APIPerformance = s.calculateAPIPerformance(apiRequests)

	return analytics, nil
}

// GetCostAnalytics returns cost analytics for a tenant
func (s *TenantAnalyticsService) GetCostAnalytics(ctx context.Context, tenantID string, timeRange string) (*TenantCostAnalytics, error) {
	// Parse time range
	startDate, endDate := s.parseTimeRange(timeRange)

	// Get tenant configuration to understand pricing
	config, err := s.repository.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}

	// Build cost analytics
	analytics := &TenantCostAnalytics{
		TenantID:    tenantID,
		TimeRange:   timeRange,
		StartDate:   startDate,
		EndDate:     endDate,
		TotalCost:   0,
		CostByType:  make(map[string]float64),
		CostTrends:  []*CostTrend{},
		Predictions: &CostPrediction{},
		Savings:     &CostSavings{},
	}

	// Calculate costs based on usage and plan
	plan := s.getTenantPlan(tenantID, config)
	analytics.TotalCost = s.calculateTotalCost(plan, analytics.TimeRange)
	analytics.CostByType = s.calculateCostByType(plan, analytics.TimeRange)

	return analytics, nil
}

// GenerateTenantReport generates a comprehensive tenant report
func (s *TenantAnalyticsService) GenerateTenantReport(ctx context.Context, tenantID string, reportType string, timeRange string) (*TenantReport, error) {
	report := &TenantReport{
		TenantID:    tenantID,
		ReportType:  reportType,
		TimeRange:   timeRange,
		GeneratedAt: time.Now(),
	}

	switch reportType {
	case "dashboard":
		dashboard, err := s.GetTenantDashboard(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		report.Dashboard = dashboard

	case "usage":
		usageAnalytics, err := s.GetUsageAnalytics(ctx, tenantID, timeRange)
		if err != nil {
			return nil, err
		}
		report.UsageAnalytics = usageAnalytics

	case "performance":
		performanceAnalytics, err := s.GetPerformanceAnalytics(ctx, tenantID, timeRange)
		if err != nil {
			return nil, err
		}
		report.PerformanceAnalytics = performanceAnalytics

	case "cost":
		costAnalytics, err := s.GetCostAnalytics(ctx, tenantID, timeRange)
		if err != nil {
			return nil, err
		}
		report.CostAnalytics = costAnalytics

	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	return report, nil
}

// Helper methods

func (s *TenantAnalyticsService) calculateHealthScore(usageStats *TenantUsageStats, errorRate float64) float64 {
	score := 100.0

	// Deduct points for high error rate
	if errorRate > 10 {
		score -= 30
	} else if errorRate > 5 {
		score -= 15
	} else if errorRate > 1 {
		score -= 5
	}

	// Deduct points for quota issues
	for _, quotaStatus := range usageStats.QuotaStatus {
		if quotaStatus.Critical {
			score -= 20
		} else if quotaStatus.Warning {
			score -= 10
		}
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score
}

func (s *TenantAnalyticsService) generateAlerts(usageStats *TenantUsageStats, errorRate float64) []string {
	alerts := []string{}

	// Error rate alerts
	if errorRate > 10 {
		alerts = append(alerts, "Critical: High error rate detected")
	} else if errorRate > 5 {
		alerts = append(alerts, "Warning: Elevated error rate")
	}

	// Quota alerts
	for resourceType, quotaStatus := range usageStats.QuotaStatus {
		if quotaStatus.Critical {
			alerts = append(alerts, fmt.Sprintf("Critical: %s quota at %.1f%%", resourceType, quotaStatus.Percent))
		} else if quotaStatus.Warning {
			alerts = append(alerts, fmt.Sprintf("Warning: %s quota at %.1f%%", resourceType, quotaStatus.Percent))
		}
	}

	return alerts
}

func (s *TenantAnalyticsService) parseTimeRange(timeRange string) (time.Time, time.Time) {
	now := time.Now()

	switch timeRange {
	case "1h":
		return now.Add(-1 * time.Hour), now
	case "24h":
		return now.Add(-24 * time.Hour), now
	case "7d":
		return now.Add(-7 * 24 * time.Hour), now
	case "30d":
		return now.Add(-30 * 24 * time.Hour), now
	case "90d":
		return now.Add(-90 * 24 * time.Hour), now
	default:
		return now.Add(-24 * time.Hour), now
	}
}

func (s *TenantAnalyticsService) parseAPIRequestEvent(event *TenantEvent) *APIRequestEvent {
	// Implementation depends on event structure
	return &APIRequestEvent{
		Timestamp:    event.Timestamp,
		Endpoint:     "",
		Method:       "",
		StatusCode:   200,
		ResponseTime: 0,
		UserID:       event.UserID,
	}
}

func (s *TenantAnalyticsService) parseErrorEvent(event *TenantEvent) *ErrorEvent {
	// Implementation depends on event structure
	return &ErrorEvent{
		Timestamp: event.Timestamp,
		Error:     "",
		Context:   event.EventData,
		UserID:    event.UserID,
	}
}

func (s *TenantAnalyticsService) parseSlowQueryEvent(event *TenantEvent) *SlowQuery {
	// Implementation depends on event structure
	return &SlowQuery{
		Timestamp: event.Timestamp,
		Query:     "",
		Duration:  0,
		Context:   event.EventData,
	}
}

func (s *TenantAnalyticsService) calculateAPIPerformance(requests []*APIRequestEvent) *APIPerformance {
	// Implementation would calculate performance metrics
	return &APIPerformance{
		TotalRequests:       len(requests),
		AverageResponseTime: 0,
		P95ResponseTime:     0,
		ErrorRate:           0,
		RequestsPerSecond:   0,
	}
}

func (s *TenantAnalyticsService) getTenantPlan(tenantID string, config *TenantConfig) TenantPlan {
	// Extract plan from tenant config or database
	return TenantPlanPro // Default
}

func (s *TenantAnalyticsService) calculateTotalCost(plan TenantPlan, timeRange string) float64 {
	// Implementation would calculate cost based on plan and usage
	return 0.0
}

func (s *TenantAnalyticsService) calculateCostByType(plan TenantPlan, timeRange string) map[string]float64 {
	// Implementation would calculate cost breakdown by resource type
	return make(map[string]float64)
}

// Data structures for analytics

type TenantDashboard struct {
	TenantID     string            `json:"tenant_id"`
	UsageStats   *TenantUsageStats `json:"usage_stats"`
	Metrics      *TenantMetrics    `json:"metrics"`
	RecentEvents []*TenantEvent    `json:"recent_events"`
	QuotaStatus  []*TenantUsage    `json:"quota_status"`
	LastUpdated  time.Time         `json:"last_updated"`
}

type TenantUsageAnalytics struct {
	TenantID    string                             `json:"tenant_id"`
	TimeRange   string                             `json:"time_range"`
	StartDate   time.Time                          `json:"start_date"`
	EndDate     time.Time                          `json:"end_date"`
	UsageByType map[string]*ResourceUsageAnalytics `json:"usage_by_type"`
	Trends      map[string][]*UsageTrend           `json:"trends"`
	Peaks       map[string]*UsagePeak              `json:"peaks"`
}

type ResourceUsageAnalytics struct {
	ResourceType string    `json:"resource_type"`
	TotalUsage   int       `json:"total_usage"`
	AverageUsage int       `json:"average_usage"`
	PeakUsage    int       `json:"peak_usage"`
	PeakTime     time.Time `json:"peak_time"`
}

type UsageTrend struct {
	Timestamp time.Time `json:"timestamp"`
	Usage     int       `json:"usage"`
}

type UsagePeak struct {
	Timestamp time.Time              `json:"timestamp"`
	Usage     int                    `json:"usage"`
	Context   map[string]interface{} `json:"context"`
}

type TenantPerformanceAnalytics struct {
	TenantID            string                          `json:"tenant_id"`
	TimeRange           string                          `json:"time_range"`
	StartDate           time.Time                       `json:"start_date"`
	EndDate             time.Time                       `json:"end_date"`
	APIPerformance      *APIPerformance                 `json:"api_performance"`
	ResourcePerformance map[string]*ResourcePerformance `json:"resource_performance"`
	Errors              []*ErrorEvent                   `json:"errors"`
	SlowQueries         []*SlowQuery                    `json:"slow_queries"`
}

type APIPerformance struct {
	TotalRequests       int     `json:"total_requests"`
	AverageResponseTime float64 `json:"average_response_time"`
	P95ResponseTime     float64 `json:"p95_response_time"`
	ErrorRate           float64 `json:"error_rate"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
}

type ResourcePerformance struct {
	ResourceType string  `json:"resource_type"`
	AvgLatency   float64 `json:"avg_latency"`
	P95Latency   float64 `json:"p95_latency"`
	Throughput   float64 `json:"throughput"`
	ErrorRate    float64 `json:"error_rate"`
}

type APIRequestEvent struct {
	Timestamp    time.Time     `json:"timestamp"`
	Endpoint     string        `json:"endpoint"`
	Method       string        `json:"method"`
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	UserID       string        `json:"user_id"`
}

type ErrorEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error"`
	Context   map[string]interface{} `json:"context"`
	UserID    string                 `json:"user_id"`
}

type SlowQuery struct {
	Timestamp time.Time              `json:"timestamp"`
	Query     string                 `json:"query"`
	Duration  time.Duration          `json:"duration"`
	Context   map[string]interface{} `json:"context"`
}

type TenantCostAnalytics struct {
	TenantID    string             `json:"tenant_id"`
	TimeRange   string             `json:"time_range"`
	StartDate   time.Time          `json:"start_date"`
	EndDate     time.Time          `json:"end_date"`
	TotalCost   float64            `json:"total_cost"`
	CostByType  map[string]float64 `json:"cost_by_type"`
	CostTrends  []*CostTrend       `json:"cost_trends"`
	Predictions *CostPrediction    `json:"predictions"`
	Savings     *CostSavings       `json:"savings"`
}

type CostTrend struct {
	Timestamp time.Time `json:"timestamp"`
	Cost      float64   `json:"cost"`
}

type CostPrediction struct {
	PredictedCost float64  `json:"predicted_cost"`
	Confidence    float64  `json:"confidence"`
	Factors       []string `json:"factors"`
}

type CostSavings struct {
	PotentialSavings float64            `json:"potential_savings"`
	Recommendations  []string           `json:"recommendations"`
	Optimizations    map[string]float64 `json:"optimizations"`
}

type TenantReport struct {
	TenantID             string                      `json:"tenant_id"`
	ReportType           string                      `json:"report_type"`
	TimeRange            string                      `json:"time_range"`
	GeneratedAt          time.Time                   `json:"generated_at"`
	Dashboard            *TenantDashboard            `json:"dashboard,omitempty"`
	UsageAnalytics       *TenantUsageAnalytics       `json:"usage_analytics,omitempty"`
	PerformanceAnalytics *TenantPerformanceAnalytics `json:"performance_analytics,omitempty"`
	CostAnalytics        *TenantCostAnalytics        `json:"cost_analytics,omitempty"`
}
