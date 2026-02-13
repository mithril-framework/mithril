package utils

import (
	"fmt"
	"strings"
)

// PrintSuccess prints message with green checkmark.
func PrintSuccess(message string) {
	fmt.Printf("\033[32m✓ %s\033[0m\n", message)
}

// PrintError prints message with red cross.
func PrintError(message string) {
	fmt.Printf("\033[31m✗ %s\033[0m\n", message)
}

// PrintWarning prints message with yellow warning.
func PrintWarning(message string) {
	fmt.Printf("\033[33m⚠ %s\033[0m\n", message)
}

// PrintInfo prints message with blue info.
func PrintInfo(message string) {
	fmt.Printf("\033[34mℹ %s\033[0m\n", message)
}

// AskInput prompts and reads a trimmed line of input.
func AskInput(prompt string) string {
	fmt.Print(prompt + ": ")
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input)
}

// AskConfirmation returns true if user answers y/yes (case-insensitive).
func AskConfirmation(prompt string) bool {
	response := AskInput(prompt + " (y/N)")
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// ShowProgress prints a progress bar for current/total with message (50-char bar).
func ShowProgress(current, total int, message string) {
	percentage := float64(current) / float64(total) * 100
	barLength := 50
	filledLength := int(percentage/100 * float64(barLength))
	bar := strings.Repeat("=", filledLength) + strings.Repeat("-", barLength-filledLength)
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)", message, bar, percentage, current, total)
	if current == total {
		fmt.Println()
	}
}
