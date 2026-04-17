package commands

import (
	"fmt"
)

func HandleSubscribers(args []string) {
	if len(args) == 0 {
		showSubscribersHelp()
		return
	}

	command := args[0]
	switch command {
	case "list":
		listSubscribers()
	case "create":
		createSubscriber(args[1:])
	case "delete":
		deleteSubscriber(args[1:])
	case "show":
		showSubscriber(args[1:])
	default:
		fmt.Printf("Unknown subscribers command: %s\n", command)
		showSubscribersHelp()
	}
}

func showSubscribersHelp() {
	fmt.Println("Subscriber Management")
	fmt.Println("Usage: telecom-cli subscribers <command> [options]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  list                    - List all subscribers")
	fmt.Println("  create <imsi> <name>    - Create a new subscriber")
	fmt.Println("  delete <imsi>           - Delete a subscriber")
	fmt.Println("  show <imsi>             - Show subscriber details")
}

func listSubscribers() {
	fmt.Println("Active Subscribers:")
	fmt.Println("IMSI          Name                Status    Balance")
	fmt.Println("----------------------------------------------------")
	fmt.Println("310260123456789 John Doe           Active    $45.67")
	fmt.Println("310260123456790 Jane Smith         Active    $123.45")
	fmt.Println("310260123456791 Bob Johnson        Inactive  $0.00")
}

func createSubscriber(args []string) {
	if len(args) < 2 {
		fmt.Println("Error: IMSI and name are required")
		fmt.Println("Usage: telecom-cli subscribers create <imsi> <name>")
		return
	}

	imsi := args[0]
	name := args[1]

	fmt.Printf("Creating subscriber: IMSI=%s, Name=%s\n", imsi, name)
	fmt.Println("Subscriber created successfully!")
}

func deleteSubscriber(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: IMSI is required")
		fmt.Println("Usage: telecom-cli subscribers delete <imsi>")
		return
	}

	imsi := args[0]
	fmt.Printf("Deleting subscriber: IMSI=%s\n", imsi)
	fmt.Println("Subscriber deleted successfully!")
}

func showSubscriber(args []string) {
	if len(args) < 1 {
		fmt.Println("Error: IMSI is required")
		fmt.Println("Usage: telecom-cli subscribers show <imsi>")
		return
	}

	imsi := args[0]
	fmt.Printf("Subscriber Details: IMSI=%s\n", imsi)
	fmt.Println("Name: John Doe")
	fmt.Println("Status: Active")
	fmt.Println("Balance: $45.67")
	fmt.Println("Data Used: 2.3GB / 10GB")
	fmt.Println("Voice Used: 45min / 500min")
	fmt.Println("SMS Used: 23 / 100")
}
