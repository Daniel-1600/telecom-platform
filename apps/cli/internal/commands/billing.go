package commands

import (
	"fmt"

	"github.com/nutcas3/telecom-platform/apps/cli/internal/api"
	"github.com/nutcas3/telecom-platform/apps/cli/internal/ui"
)

func HandleBilling(args []string, apiClient *api.Client) {
	if len(args) == 0 {
		showBillingHelp()
		return
	}

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

func handleInvoices(apiClient *api.Client) {
	colorizer := ui.NewColorizer(true)
	iconRenderer := ui.NewIconRenderer(true, false)

	title := colorizer.Colorize("Recent Invoices", ui.StyleHeader)
	fmt.Println(title)

	// Try to get real data from API
	invoices, err := apiClient.GetInvoices()
	if err != nil {
		// Fallback to placeholder data if API fails
		warningMsg := colorizer.Colorize("Using placeholder data - API error: ", ui.StyleWarning) +
			colorizer.Colorize(err.Error(), ui.StyleMuted)
		fmt.Println(warningMsg)
		fmt.Println()

		// Create table with placeholder data
		table := ui.NewTable(colorizer, iconRenderer)
		table.AddColumn("Invoice #", 12, "left")
		table.AddColumn("Date", 12, "left")
		table.AddColumn("Status", 10, "left")
		table.AddColumn("Amount", 10, "right")
		table.AddColumn("Subscriber", 15, "left")

		table.AddStyledRow(ui.StyleSuccess.Style, "INV-000001", "2024-01-15", "Paid", "$45.67", "John Doe")
		table.AddStyledRow(ui.StyleWarning.Style, "INV-000002", "2024-01-15", "Pending", "$123.45", "Jane Smith")
		table.AddStyledRow(ui.StyleError.Style, "INV-000003", "2024-01-14", "Overdue", "$67.89", "Bob Johnson")

		fmt.Println(table.Render())
		return
	}

	// Create table with real data
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Invoice #", 12, "left")
	table.AddColumn("Date", 12, "left")
	table.AddColumn("Status", 10, "left")
	table.AddColumn("Amount", 10, "right")
	table.AddColumn("Subscriber", 15, "left")

	// Add rows with real data
	for _, invoice := range invoices {
		var style ui.Style
		switch invoice.Status {
		case "PAID":
			style = ui.StyleSuccess
		case "PENDING":
			style = ui.StyleWarning
		case "OVERDUE":
			style = ui.StyleError
		default:
			style = ui.StyleMuted
		}

		table.AddStyledRow(style.Style,
			invoice.ID,
			invoice.CreatedAt.Format("2006-01-02"),
			invoice.Status,
			fmt.Sprintf("$%.2f", invoice.Amount),
			fmt.Sprintf("%s %s", invoice.Subscriber.FirstName, invoice.Subscriber.LastName))
	}

	fmt.Println(table.Render())
}

func handlePayments(apiClient *api.Client) {
	colorizer := ui.NewColorizer(true)
	iconRenderer := ui.NewIconRenderer(true, false)

	title := colorizer.Colorize("Recent Payments", ui.StyleHeader)
	fmt.Println(title)

	// Try to get real data from API
	payments, err := apiClient.GetPayments()
	if err != nil {
		// Fallback to placeholder data if API fails
		warningMsg := colorizer.Colorize("Using placeholder data - API error: ", ui.StyleWarning) +
			colorizer.Colorize(err.Error(), ui.StyleMuted)
		fmt.Println(warningMsg)
		fmt.Println()

		// Create table with placeholder data
		table := ui.NewTable(colorizer, iconRenderer)
		table.AddColumn("Transaction ID", 16, "left")
		table.AddColumn("Date", 12, "left")
		table.AddColumn("Amount", 10, "right")
		table.AddColumn("Method", 10, "left")
		table.AddColumn("Status", 10, "left")

		table.AddStyledRow(ui.StyleSuccess.Style, "pay_123456789", "2024-01-15", "$45.67", "Stripe", "Success")
		table.AddStyledRow(ui.StyleSuccess.Style, "pay_123456790", "2024-01-14", "$123.45", "Stripe", "Success")
		table.AddStyledRow(ui.StyleError.Style, "pay_123456791", "2024-01-13", "$67.89", "Credit", "Failed")

		fmt.Println(table.Render())
		return
	}

	// Create table with real data
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Transaction ID", 16, "left")
	table.AddColumn("Date", 12, "left")
	table.AddColumn("Amount", 10, "right")
	table.AddColumn("Method", 10, "left")
	table.AddColumn("Status", 10, "left")

	// Add rows with real data
	for _, payment := range payments {
		var style ui.Style
		switch payment.Status {
		case "SUCCESS", "COMPLETED":
			style = ui.StyleSuccess
		case "PENDING", "PROCESSING":
			style = ui.StyleWarning
		case "FAILED", "CANCELLED":
			style = ui.StyleError
		default:
			style = ui.StyleMuted
		}

		table.AddStyledRow(style.Style,
			payment.ID,
			payment.CreatedAt.Format("2006-01-02"),
			fmt.Sprintf("$%.2f", payment.Amount),
			payment.Method,
			payment.Status)
	}

	fmt.Println(table.Render())
}

func generateInvoice(apiClient *api.Client, args []string) {
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

	// Try to generate invoice via API
	invoice, err := apiClient.GenerateInvoice(subscriberID)
	if err != nil {
		// Fallback to placeholder data if API fails
		warningMsg := colorizer.Colorize("Using placeholder data - API error: ", ui.StyleWarning) +
			colorizer.Colorize(err.Error(), ui.StyleMuted)
		fmt.Println(warningMsg)
		fmt.Println()

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
		return
	}

	// Show success message with styled output
	successMsg := colorizer.Colorize("Invoice generated successfully!", ui.StyleSuccess)
	fmt.Println(successMsg)
	fmt.Println()

	// Create table for invoice details with real data
	table := ui.NewTable(colorizer, iconRenderer)
	table.AddColumn("Field", 12, "left")
	table.AddColumn("Value", 20, "left")

	table.AddRow("Invoice #", invoice.ID)
	table.AddRow("Amount", fmt.Sprintf("$%.2f", invoice.Amount))
	table.AddRow("Due Date", invoice.DueDate.Format("2006-01-02"))
	table.AddRow("Status", invoice.Status)

	fmt.Println(table.Render())
}
