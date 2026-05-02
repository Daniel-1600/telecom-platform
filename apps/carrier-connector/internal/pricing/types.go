package pricing

import (
	"time"

	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/tenant"
)

// PricingRule represents a dynamic pricing rule
type PricingRule struct {
	ID          string         `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	TenantID    string         `json:"tenant_id" gorm:"index;not null"`
	Type        RuleType       `json:"type" gorm:"not null"`
	Priority    int            `json:"priority" gorm:"default:0"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	Conditions  RuleConditions `json:"conditions" gorm:"serializer:json"`
	Actions     RuleActions    `json:"actions" gorm:"serializer:json"`
	Metadata    map[string]any `json:"metadata" gorm:"serializer:json"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// RuleType defines the type of pricing rule
type RuleType string

const (
	RuleTypePercentageDiscount RuleType = "percentage_discount"
	RuleTypeFixedDiscount      RuleType = "fixed_discount"
	RuleTypeMultiplier         RuleType = "multiplier"
	RuleTypeTieredPricing      RuleType = "tiered_pricing"
	RuleTypeDynamicPricing     RuleType = "dynamic_pricing"
	RuleTypeConditionalPricing RuleType = "conditional_pricing"
)

// RuleConditions defines when a rule applies
type RuleConditions struct {
	TimeRange    *TimeRange    `json:"time_range,omitempty"`
	Geography    []string      `json:"geography,omitempty"`
	CustomerType []string      `json:"customer_type,omitempty"`
	Volume       *VolumeRange  `json:"volume,omitempty"`
	UsagePattern *UsagePattern `json:"usage_pattern,omitempty"`
}

// TimeRange defines time-based conditions
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Days  []string `json:"days"` // ["monday", "tuesday", ...]
}

// VolumeRange defines volume-based conditions
type VolumeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// UsagePattern defines usage-based conditions
type UsagePattern struct {
	PeakHours    []string `json:"peak_hours"`
	OffPeakHours []string `json:"off_peak_hours"`
}

// RuleActions defines what happens when a rule applies
type RuleActions struct {
	AdjustmentType AdjustmentType `json:"adjustment_type"`
	Value         float64         `json:"value"`
	NewPrice      *float64        `json:"new_price,omitempty"`
	Limit         *float64        `json:"limit,omitempty"`
}

// AdjustmentType defines how pricing is adjusted
type AdjustmentType string

const (
	AdjustmentTypePercentage AdjustmentType = "percentage"
	AdjustmentTypeFixed      AdjustmentType = "fixed"
	AdjustmentTypeMultiply   AdjustmentType = "multiply"
	AdjustmentTypeOverride   AdjustmentType = "override"
)

// PricingContext contains context for pricing calculations
type PricingContext struct {
	TenantID     string                 `json:"tenant_id"`
	CustomerID   string                 `json:"customer_id"`
	ProductID    string                 `json:"product_id"`
	BasePrice    float64                `json:"base_price"`
	Currency     string                 `json:"currency"`
	Quantity     int                    `json:"quantity"`
	Location     string                 `json:"location"`
	Time         time.Time              `json:"time"`
	Metadata     map[string]any         `json:"metadata"`
	TenantCtx    *tenant.TenantContext  `json:"tenant_context,omitempty"`
}

// PricingResult contains the result of pricing calculations
type PricingResult struct {
	OriginalPrice float64         `json:"original_price"`
	AdjustedPrice float64         `json:"adjusted_price"`
	FinalPrice    float64         `json:"final_price"`
	Currency      string          `json:"currency"`
	Discount      float64         `json:"discount"`
	AppliedRules  []AppliedRule   `json:"applied_rules"`
	Metadata      map[string]any  `json:"metadata"`
}

// AppliedRule represents a rule that was applied
type AppliedRule struct {
	RuleID    string  `json:"rule_id"`
	RuleName  string  `json:"rule_name"`
	Type      string  `json:"type"`
	Adjustment float64 `json:"adjustment"`
}

// PricingFilter defines filtering options for pricing rules
type PricingFilter struct {
	TenantID  string     `json:"tenant_id,omitempty"`
	Type      string     `json:"type,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
	Priority  *int       `json:"priority,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// PricingAnalytics contains analytics data
type PricingAnalytics struct {
	TotalRules      int                    `json:"total_rules"`
	ActiveRules     int                    `json:"active_rules"`
	RulesByType     map[string]int         `json:"rules_by_type"`
	UsageByRule     map[string]int64       `json:"usage_by_rule"`
	DiscountStats   DiscountStatistics     `json:"discount_stats"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// DiscountStatistics contains discount statistics
type DiscountStatistics struct {
	TotalDiscounts     int64   `json:"total_discounts"`
	AverageDiscount    float64 `json:"average_discount"`
	LargestDiscount    float64 `json:"largest_discount"`
	SmallestDiscount   float64 `json:"smallest_discount"`
	TotalDiscountValue float64 `json:"total_discount_value"`
}
