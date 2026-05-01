package integration

import (
	"log"
	"time"

	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/config"
	"github.com/nutcas3/telecom-platform/apps/carrier-connector/internal/smdp"
)

// SetupCarriers configures default carriers for demonstration
func (si *SelectionIntegration) SetupCarriers() error {
	// Configure sample carriers with different characteristics
	carriers := []*smdp.Carrier{
		{
			ID:          "att-us",
			Name:        "AT&T US",
			MCC:         "310",
			MNC:         "410",
			CountryCode: "US",
			IsActive:    true,
			Priority:    90,
			ES2Config: &config.ES2Config{
				BaseURL:                  "https://es2plus.att.com",
				APIKey:                   "demo-key-att",
				InsecureSkipVerify:       false,
				FunctionalityRequesterID: "telecom-platform",
			},
			Capabilities: &smdp.CarrierCapabilities{
				SupportedProfileTypes: []string{"operational", "testing"},
				Features:              []string{"bulk_download", "remote_provisioning"},
				MaxConcurrentRequests: 100,
			},
			Metrics: &smdp.CarrierMetrics{
				TotalRequests:       1000,
				SuccessfulRequests:  980,
				FailedRequests:      20,
				AverageResponseTime: 150 * time.Millisecond,
				RequestRate:         10.5,
			},
		},
		{
			ID:          "verizon-us",
			Name:        "Verizon US",
			MCC:         "311",
			MNC:         "480",
			CountryCode: "US",
			IsActive:    true,
			Priority:    85,
			ES2Config: &config.ES2Config{
				BaseURL:                  "https://es2plus.verizon.com",
				APIKey:                   "demo-key-verizon",
				InsecureSkipVerify:       false,
				FunctionalityRequesterID: "telecom-platform",
			},
			Capabilities: &smdp.CarrierCapabilities{
				SupportedProfileTypes: []string{"operational", "testing"},
				Features:              []string{"bulk_download"},
				MaxConcurrentRequests: 80,
			},
			Metrics: &smdp.CarrierMetrics{
				TotalRequests:       800,
				SuccessfulRequests:  790,
				FailedRequests:      10,
				AverageResponseTime: 120 * time.Millisecond,
				RequestRate:         8.2,
			},
		},
		{
			ID:          "tmobile-de",
			Name:        "T-Mobile Germany",
			MCC:         "262",
			MNC:         "01",
			CountryCode: "DE",
			IsActive:    true,
			Priority:    75,
			ES2Config: &config.ES2Config{
				BaseURL:                  "https://es2plus.t-mobile.de",
				APIKey:                   "demo-key-tmobile",
				InsecureSkipVerify:       false,
				FunctionalityRequesterID: "telecom-platform",
			},
			Capabilities: &smdp.CarrierCapabilities{
				SupportedProfileTypes: []string{"operational"},
				Features:              []string{"remote_provisioning"},
				MaxConcurrentRequests: 60,
			},
			Metrics: &smdp.CarrierMetrics{
				TotalRequests:       600,
				SuccessfulRequests:  570,
				FailedRequests:      30,
				AverageResponseTime: 200 * time.Millisecond,
				RequestRate:         6.8,
			},
		},
		{
			ID:          "orange-fr",
			Name:        "Orange France",
			MCC:         "208",
			MNC:         "01",
			CountryCode: "FR",
			IsActive:    true,
			Priority:    70,
			ES2Config: &config.ES2Config{
				BaseURL:                  "https://es2plus.orange.fr",
				APIKey:                   "demo-key-orange",
				InsecureSkipVerify:       false,
				FunctionalityRequesterID: "telecom-platform",
			},
			Capabilities: &smdp.CarrierCapabilities{
				SupportedProfileTypes: []string{"operational", "testing"},
				Features:              []string{},
				MaxConcurrentRequests: 50,
			},
			Metrics: &smdp.CarrierMetrics{
				TotalRequests:       400,
				SuccessfulRequests:  380,
				FailedRequests:      20,
				AverageResponseTime: 180 * time.Millisecond,
				RequestRate:         4.5,
			},
		},
	}

	// Add carriers to the manager
	for _, carrier := range carriers {
		if err := si.manager.AddCarrier(carrier); err != nil {
			return err
		}
	}

	log.Printf("Added %d carriers to the selection manager", len(carriers))
	return nil
}
