package output

import (
	"fmt"
	"os"
)

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
)

func Header(msg string) {
	fmt.Printf("%s%s%s\n", Bold, msg, Reset)
}

func Info(msg string) {
	fmt.Printf("  %s%s%s\n", Cyan, msg, Reset)
}

func Success(msg string) {
	fmt.Printf("  %s✓%s %s\n", Green, Reset, msg)
}

func Warning(msg string) {
	fmt.Fprintf(os.Stderr, "  %s⚠%s %s\n", Yellow, Reset, msg)
}

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "  %s✗%s %s\n", Red, Reset, msg)
}

func Detail(msg string) {
	fmt.Printf("    %s\n", msg)
}
