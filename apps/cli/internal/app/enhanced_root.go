package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nutcas3/telecom-platform/apps/cli/internal/commands"
	"github.com/nutcas3/telecom-platform/apps/cli/internal/types"
)

type EnhancedCLI struct {
	config *types.CLIConfig
}

func NewEnhancedCLI() *EnhancedCLI {
	return &EnhancedCLI{
		config: &types.CLIConfig{
			APIEndpoint: "http://localhost:8000",
			APIToken:    "",
			Profile:     "default",
			Verbose:     false,
			NoColor:     false,
			Theme:       "default",
		},
	}
}

func (cli *EnhancedCLI) Run(args []string) error {
	if len(args) < 2 {
		cli.showHelp()
		return nil
	}

	command := args[1]
	commandArgs := args[2:]

	switch command {
	case "dashboard":
		return cli.runDashboard()
	case "subscribers":
		return commands.HandleSubscribersEnhanced(commandArgs, cli.config)
	case "services":
		return commands.HandleServicesEnhanced(commandArgs, cli.config)
	case "billing":
		return commands.HandleBillingEnhanced(commandArgs, cli.config)
	case "monitoring":
		return commands.HandleMonitoringEnhanced(commandArgs, cli.config)
	case "config":
		return commands.HandleConfigEnhanced(commandArgs, cli.config)
	case "deploy":
		return commands.HandleDeployEnhanced(commandArgs, cli.config)
	case "plugins":
		return commands.HandlePluginsEnhanced(commandArgs, cli.config)
	case "automation":
		return commands.HandleAutomationEnhanced(commandArgs, cli.config)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		cli.showHelp()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func (cli *EnhancedCLI) runDashboard() error {
	p := tea.NewProgram(NewDashboard())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running dashboard: %w", err)
	}
	return nil
}

func (cli *EnhancedCLI) showHelp() {
	fmt.Println("Telecom Platform CLI - Enhanced")
	fmt.Println("Usage: telecom-cli <command> [options]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  dashboard        - Interactive dashboard for platform overview")
	fmt.Println("  subscribers      - Subscriber management")
	fmt.Println("  services         - Service management")
	fmt.Println("  billing          - Billing and invoice management")
	fmt.Println("  monitoring       - Monitoring and metrics")
	fmt.Println("  config           - Configuration management")
	fmt.Println("  deploy           - Deployment management")
	fmt.Println("  plugins          - Plugin management")
	fmt.Println("  automation       - Automation and scripting")
	fmt.Println()
	fmt.Println("Global options:")
	fmt.Println("  --endpoint <url>    API endpoint (default: http://localhost:8000)")
	fmt.Println("  --token <token>     API authentication token")
	fmt.Println("  --profile <name>     Configuration profile")
	fmt.Println("  --verbose           Enable verbose output")
	fmt.Println("  --no-color          Disable color output")
	fmt.Println()
	fmt.Println("Use 'telecom-cli <command> --help' for more information")
}

func (cli *EnhancedCLI) parseConfig(args []string) error {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--endpoint":
			if i+1 < len(args) {
				cli.config.APIEndpoint = args[i+1]
				i++
			}
		case "--token":
			if i+1 < len(args) {
				cli.config.APIToken = args[i+1]
				i++
			}
		case "--profile":
			if i+1 < len(args) {
				cli.config.Profile = args[i+1]
				i++
			}
		case "--verbose":
			cli.config.Verbose = true
		case "--no-color":
			cli.config.NoColor = true
		}
	}
	return nil
}

func Main() {
	cli := NewEnhancedCLI()

	// Parse global options
	if err := cli.parseConfig(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing options: %v\n", err)
		os.Exit(1)
	}

	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
