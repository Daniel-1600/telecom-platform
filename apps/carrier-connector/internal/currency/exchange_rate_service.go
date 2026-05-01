package currency

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// ExchangeRateService manages currency exchange rates
type ExchangeRateService struct {
	repository   Repository
	logger       *logrus.Logger
	providers    []ExchangeRateProvider
	baseCurrency string
}

// NewExchangeRateService creates a new exchange rate service
func NewExchangeRateService(repository Repository, logger *logrus.Logger, baseCurrency string) *ExchangeRateService {
	return &ExchangeRateService{
		repository:   repository,
		logger:       logger,
		providers:    make([]ExchangeRateProvider, 0),
		baseCurrency: baseCurrency,
	}
}

// AddProvider adds an exchange rate provider
func (s *ExchangeRateService) AddProvider(provider ExchangeRateProvider) {
	s.providers = append(s.providers, provider)
}

// GetExchangeRate gets the current exchange rate between two currencies
func (s *ExchangeRateService) GetExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*ExchangeRate, error) {
	if fromCurrency == toCurrency {
		return &ExchangeRate{
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			Rate:         1.0,
			Source:       "direct",
			ValidFrom:    time.Now(),
			IsActive:     true,
		}, nil
	}

	// Try to get from repository first
	rate, err := s.repository.GetLatestExchangeRate(ctx, fromCurrency, toCurrency)
	if err == nil {
		return rate, nil
	}

	// If not found, try to get from providers
	for _, provider := range s.providers {
		providerRate, err := provider.GetRate(ctx, fromCurrency, toCurrency)
		if err == nil {
			// Save to repository
			newRate := &ExchangeRate{
				ID:           fmt.Sprintf("%s_%s_%d", fromCurrency, toCurrency, time.Now().Unix()),
				FromCurrency: fromCurrency,
				ToCurrency:   toCurrency,
				Rate:         providerRate,
				Source:       "provider",
				ValidFrom:    time.Now(),
				IsActive:     true,
			}

			if err := s.repository.CreateExchangeRate(ctx, newRate); err != nil {
				s.logger.WithError(err).Error("Failed to save exchange rate")
			}

			return newRate, nil
		}
	}

	return nil, fmt.Errorf("exchange rate not found: %s to %s", fromCurrency, toCurrency)
}

// ConvertAmount converts an amount from one currency to another
func (s *ExchangeRateService) ConvertAmount(ctx context.Context, amount float64, fromCurrency, toCurrency string) (*CurrencyConversionResponse, error) {
	rate, err := s.GetExchangeRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange rate: %w", err)
	}

	convertedAmount := amount * rate.Rate

	return &CurrencyConversionResponse{
		OriginalAmount:    amount,
		OriginalCurrency:  fromCurrency,
		ConvertedAmount:   convertedAmount,
		ConvertedCurrency: toCurrency,
		ExchangeRate:      rate.Rate,
		ConvertedAt:       time.Now(),
	}, nil
}

// RefreshRates refreshes exchange rates from all providers
func (s *ExchangeRateService) RefreshRates(ctx context.Context) error {
	s.logger.Info("Refreshing exchange rates")

	for _, provider := range s.providers {
		if err := provider.RefreshRates(ctx); err != nil {
			s.logger.WithError(err).Error("Failed to refresh rates from provider")
			continue
		}
	}

	s.logger.Info("Exchange rates refreshed successfully")
	return nil
}

// GetRateHistory gets historical exchange rates
func (s *ExchangeRateService) GetRateHistory(ctx context.Context, fromCurrency, toCurrency string, days int) ([]*ExchangeRate, error) {
	filter := &ExchangeRateFilter{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		IsValid:      &[]bool{false}[0], // Include historical rates
		Limit:        days,
		SortBy:       "valid_from",
		SortOrder:    "desc",
	}

	rates, err := s.repository.ListExchangeRates(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate history: %w", err)
	}

	return rates, nil
}

// UpdateExchangeRate updates an exchange rate
func (s *ExchangeRateService) UpdateExchangeRate(ctx context.Context, rate *ExchangeRate) error {
	// Validate rate
	if rate.Rate <= 0 {
		return fmt.Errorf("invalid exchange rate: must be positive")
	}

	if rate.FromCurrency == rate.ToCurrency {
		return fmt.Errorf("invalid currency pair: from and to currencies cannot be the same")
	}

	// Set validity
	now := time.Now()
	rate.ValidFrom = now
	rate.IsActive = true

	// Deactivate old rates
	filter := &ExchangeRateFilter{
		FromCurrency: rate.FromCurrency,
		ToCurrency:   rate.ToCurrency,
		IsValid:      &[]bool{true}[0],
	}

	oldRates, err := s.repository.ListExchangeRates(ctx, filter)
	if err == nil {
		for _, oldRate := range oldRates {
			oldRate.IsActive = false
			if err := s.repository.UpdateExchangeRate(ctx, oldRate); err != nil {
				s.logger.WithError(err).Error("Failed to deactivate old exchange rate")
			}
		}
	}

	// Create new rate
	rate.ID = fmt.Sprintf("%s_%s_%d", rate.FromCurrency, rate.ToCurrency, time.Now().Unix())
	if err := s.repository.CreateExchangeRate(ctx, rate); err != nil {
		return fmt.Errorf("failed to create exchange rate: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"from_currency": rate.FromCurrency,
		"to_currency":   rate.ToCurrency,
		"rate":          rate.Rate,
		"source":        rate.Source,
	}).Info("Exchange rate updated")

	return nil
}

// GetSupportedCurrencies gets all supported currencies
func (s *ExchangeRateService) GetSupportedCurrencies(ctx context.Context) ([]*Currency, error) {
	filter := &CurrencyFilter{
		IsActive: &[]bool{true}[0],
	}

	currencies, err := s.repository.ListCurrencies(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get supported currencies: %w", err)
	}

	return currencies, nil
}

// ValidateCurrencyPair validates if a currency pair is supported
func (s *ExchangeRateService) ValidateCurrencyPair(ctx context.Context, fromCurrency, toCurrency string) error {
	// Check if currencies exist
	_, err := s.repository.GetCurrency(ctx, fromCurrency)
	if err != nil {
		return fmt.Errorf("unsupported from currency: %s", fromCurrency)
	}

	_, err = s.repository.GetCurrency(ctx, toCurrency)
	if err != nil {
		return fmt.Errorf("unsupported to currency: %s", toCurrency)
	}

	// Check if exchange rate exists or can be obtained
	_, err = s.GetExchangeRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		return fmt.Errorf("no exchange rate available: %s to %s", fromCurrency, toCurrency)
	}

	return nil
}
