package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nutcas3/telecom-platform/apps/cli/internal/config"
)

// Client represents the API client for the telecom platform
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.APIEndpoint,
		apiKey:  cfg.APIToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Invoice represents an invoice from the API
type Invoice struct {
	ID           string    `json:"id"`
	SubscriberID string    `json:"subscriber_id"`
	Amount       float64   `json:"amount"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	DueDate      time.Time `json:"due_date"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	Subscriber   Subscriber `json:"subscriber"`
}

// Payment represents a payment from the API
type Payment struct {
	ID           string    `json:"id"`
	InvoiceID    string    `json:"invoice_id"`
	Amount       float64   `json:"amount"`
	Method       string    `json:"method"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// Subscriber represents a subscriber from the API
type Subscriber struct {
	ID        uint   `json:"id"`
	IMSI      string `json:"imsi"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Status    string `json:"status"`
}

// GetInvoices retrieves invoices from the API
func (c *Client) GetInvoices() ([]Invoice, error) {
	url := fmt.Sprintf("%s/api/v1/invoices", c.baseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response struct {
		Data []Invoice `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Data, nil
}

// GetPayments retrieves payments from the API
func (c *Client) GetPayments() ([]Payment, error) {
	url := fmt.Sprintf("%s/api/v1/payments", c.baseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response struct {
		Data []Payment `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Data, nil
}

// GenerateInvoiceRequest represents a request to generate an invoice
type GenerateInvoiceRequest struct {
	SubscriberID string `json:"subscriber_id"`
}

// GenerateInvoiceResponse represents a response from generating an invoice
type GenerateInvoiceResponse struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	DueDate   time.Time `json:"due_date"`
}

// GenerateInvoice generates a new invoice for a subscriber
func (c *Client) GenerateInvoice(subscriberID string) (*GenerateInvoiceResponse, error) {
	url := fmt.Sprintf("%s/api/v1/invoices/generate", c.baseURL)
	
	request := GenerateInvoiceRequest{
		SubscriberID: subscriberID,
	}
	
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response GenerateInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &response, nil
}

// IsConnected checks if the API server is reachable
func (c *Client) IsConnected() bool {
	url := fmt.Sprintf("%s/health", c.baseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}
