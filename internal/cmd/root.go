package cmd

import (
	"fmt"
	"os"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/output"
)

// Execute is the main entry point for the CLI.
func Execute(version string) {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	subcmd := os.Args[1]

	switch subcmd {
	case "--version", "-v":
		fmt.Println("gitrepoforge " + version)
	case "validate":
		runValidate(version, os.Args[2:])
	case "apply":
		runApply(version, os.Args[2:])
	case "report":
		runReport(version, os.Args[2:])
	case "--help", "-h", "help":
		printHelp()
	default:
		output.Error("Unknown command: " + subcmd)
		fmt.Println()
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	output.Header("gitrepoforge - Standardize and audit Git repositories")
	fmt.Println()
	fmt.Printf("  %sUsage:%s gitrepoforge <command> [flags]\n", output.Bold, output.Reset)
	fmt.Println()
	fmt.Printf("  %sCommands:%s\n", output.Bold, output.Reset)
	fmt.Printf("    %svalidate%s      Audit repos against desired state (no changes)\n", output.Cyan, output.Reset)
	fmt.Printf("    %sapply%s         Apply desired state changes to repos\n", output.Cyan, output.Reset)
	fmt.Printf("    %sreport%s        Generate a markdown report of proposed changes\n", output.Cyan, output.Reset)
	fmt.Println()
	fmt.Printf("  %sFlags:%s\n", output.Bold, output.Reset)
	fmt.Printf("    %s--repo <name>%s   Target a single repo\n", output.Cyan, output.Reset)
	fmt.Printf("    %s--json%s          Output in JSON format\n", output.Cyan, output.Reset)
	fmt.Printf("    %s--version, -v%s   Print version\n", output.Cyan, output.Reset)
	fmt.Printf("    %s--help, -h%s      Show this help message\n", output.Cyan, output.Reset)
	fmt.Println()
}
