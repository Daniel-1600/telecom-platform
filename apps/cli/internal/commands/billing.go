package commands

import (
	"fmt"

	"github.com/nutcas3/telecom-platform/apps/cli/internal/api"
	"github.com/nutcas3/telecom-platform/apps/cli/internal/config"
	"github.com/nutcas3/telecom-platform/apps/cli/internal/ui"
)

func HandleBilling(args []string, apiClient *api.Client) {
	if len(args) == 0 {
		showBillingHelp()
		return
	}

	// Initialize API client
	cfg, err := config.LoadConfig()
	if err != nil {
		colorizer := ui.NewColorizer(true)
		errorMsg := colorizer.Colorize("Failed to load configuration: ", ui.StyleError) +
			colorizer.Colorize(err.Error(), ui.StyleMuted)
		fmt.Println(errorMsg)
		return
	}

	apiClient := api.NewClient(cfg)

	// Check API connectivity
	if !apiClient.IsConnected() {
		colorizer := ui.NewColorizer(true)
		warningMsg := colorizer.Colorize("Warning: Could not connect to API server. Using placeholder data.", ui.StyleWarning)
		fmt.Println(warningMsg)
		fmt.Println()
	}

	command := args[0]
	switch command {
	case "invoices":
		handleInvoices(apiClient)
	case "payments":
		handlePayments(apiClient)
	case "generate":
		generateInvoice(apiClient, args[1:])
	default:
		colorizer := ui.NewColorizer(true)
		errorMsg := colorizer.Colorize("Unknown billing command: ", ui.StyleError) +
			colorizer.Colorize(command, ui.StyleArgument)
		fmt.Println(errorMsg)
		showBillingHelp()
	}
}

func showBillingHelp() {
	colorizer := ui.NewColorizer(true)
	iconRenderer := ui.NewIconRenderer(true, false)

	title := colorizer.Colorize("Billing Management", ui.StyleHeader)
	usage := colorizer.Colorize("Usage: telecom-cli billing <command> [options]", ui.StyleMuted)

	fmt.Println(title)
	fmt.Println(usage)
	fmt.Println()

	// Create table for commands
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Command", 25, "left")
	table.AddColumn("Description", 40, "left")

	table.AddRow("invoices", "List all invoices")
	table.AddRow("payments", "List all payments")
	table.AddRow("generate <subscriber>", "Generate invoice for subscriber")

	fmt.Println(colorizer.Colorize("Available commands:", ui.StyleHeader))
	fmt.Println(table.Render())
}

func handleInvoices(args []string) {
	colorizer := ui.NewColorizer(true)
	iconRenderer := ui.NewIconRenderer(true, false)

	title := colorizer.Colorize("Recent Invoices", ui.StyleHeader)
	fmt.Println(title)

	// Create table for invoices
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Invoice #", 12, "left")
	table.AddColumn("Date", 12, "left")
	table.AddColumn("Status", 10, "left")
	table.AddColumn("Amount", 10, "right")
	table.AddColumn("Subscriber", 15, "left")

	// Add rows with status styling
	table.AddStyledRow(ui.StyleSuccess.Style, "INV-000001", "2024-01-15", "Paid", "$45.67", "John Doe")
	table.AddStyledRow(ui.StyleWarning.Style, "INV-000002", "2024-01-15", "Pending", "$123.45", "Jane Smith")
	table.AddStyledRow(ui.StyleError.Style, "INV-000003", "2024-01-14", "Overdue", "$67.89", "Bob Johnson")

	fmt.Println(table.Render())
}

func handlePayments(args []string) {
	colorizer := ui.NewColorizer(true)
	iconRenderer := ui.NewIconRenderer(true, false)

	title := colorizer.Colorize("Recent Payments", ui.StyleHeader)
	fmt.Println(title)

	// Create table for payments
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Transaction ID", 16, "left")
	table.AddColumn("Date", 12, "left")
	table.AddColumn("Amount", 10, "right")
	table.AddColumn("Method", 10, "left")
	table.AddColumn("Status", 10, "left")

	// Add rows with status styling
	table.AddStyledRow(ui.StyleSuccess.Style, "pay_123456789", "2024-01-15", "$45.67", "Stripe", "Success")
	table.AddStyledRow(ui.StyleSuccess.Style, "pay_123456790", "2024-01-14", "$123.45", "Stripe", "Success")
	table.AddStyledRow(ui.StyleError.Style, "pay_123456791", "2024-01-13", "$67.89", "Credit", "Failed")

	fmt.Println(table.Render())
}

func generateInvoice(args []string) {
	colorizer := ui.NewColorizer(true)
	iconRenderer := ui.NewIconRenderer(true, false)

	if len(args) < 1 {
		errorMsg := colorizer.Colorize("Error: Subscriber ID is required", ui.StyleError)
		usageMsg := colorizer.Colorize("Usage: telecom-cli billing generate <subscriber_id>", ui.StyleMuted)
		fmt.Println(errorMsg)
		fmt.Println(usageMsg)
		return
	}

	subscriberID := args[0]

	// Show generation message
	generatingMsg := colorizer.Colorize("Generating invoice for subscriber: ", ui.StyleInfo) +
		colorizer.Colorize(subscriberID, ui.StyleArgument)
	fmt.Println(generatingMsg)

	// Show success message with styled output
	successMsg := colorizer.Colorize("Invoice generated successfully!", ui.StyleSuccess)
	fmt.Println(successMsg)
	fmt.Println()

	// Create table for invoice details
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Field", 12, "left")
	table.AddColumn("Value", 20, "left")

	table.AddRow("Invoice #", "INV-000004")
	table.AddRow("Amount", "$45.67")
	table.AddRow("Due Date", "2024-02-15")

	fmt.Println(table.Render())
}
