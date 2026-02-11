package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// InputProvider provides input functionality for CLI commands
type InputProvider struct {
	reader *bufio.Reader
}

// NewInputProvider creates a new input provider
func NewInputProvider() *InputProvider {
	return &InputProvider{
		reader: bufio.NewReader(os.Stdin),
	}
}

// ReadString reads a string from input
func (ip *InputProvider) ReadString(prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := ip.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

// ReadStringWithDefault reads a string with a default value
func (ip *InputProvider) ReadStringWithDefault(prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Print(prompt)
	}
	
	input, err := ip.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// ReadInt reads an integer from input
func (ip *InputProvider) ReadInt(prompt string) (int, error) {
	input, err := ip.ReadString(prompt)
	if err != nil {
		return 0, err
	}
	
	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return value, nil
}

// ReadIntWithDefault reads an integer with a default value
func (ip *InputProvider) ReadIntWithDefault(prompt string, defaultValue int) (int, error) {
	if defaultValue != 0 {
		fmt.Printf("%s [%d]: ", prompt, defaultValue)
	} else {
		fmt.Print(prompt)
	}
	
	input, err := ip.reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	
	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return value, nil
}

// ReadBool reads a boolean from input
func (ip *InputProvider) ReadBool(prompt string) (bool, error) {
	input, err := ip.ReadString(prompt)
	if err != nil {
		return false, err
	}
	
	input = strings.ToLower(input)
	switch input {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", input)
	}
}

// ReadBoolWithDefault reads a boolean with a default value
func (ip *InputProvider) ReadBoolWithDefault(prompt string, defaultValue bool) (bool, error) {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}
	
	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	input, err := ip.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	
	input = strings.ToLower(input)
	switch input {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", input)
	}
}

// ReadChoice reads a choice from a list of options
func (ip *InputProvider) ReadChoice(prompt string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices provided")
	}
	
	fmt.Println(prompt)
	for i, choice := range choices {
		fmt.Printf("  %d. %s\n", i+1, choice)
	}
	
	input, err := ip.ReadString("Enter your choice: ")
	if err != nil {
		return "", err
	}
	
	index, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid choice: %w", err)
	}
	
	if index < 1 || index > len(choices) {
		return "", fmt.Errorf("choice must be between 1 and %d", len(choices))
	}
	
	return choices[index-1], nil
}

// ReadChoiceWithDefault reads a choice with a default value
func (ip *InputProvider) ReadChoiceWithDefault(prompt string, choices []string, defaultValue string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices provided")
	}
	
	// Find default index
	defaultIndex := -1
	for i, choice := range choices {
		if choice == defaultValue {
			defaultIndex = i
			break
		}
	}
	
	fmt.Println(prompt)
	for i, choice := range choices {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Printf("  %s %d. %s\n", marker, i+1, choice)
	}
	
	input, err := ip.ReadString("Enter your choice: ")
	if err != nil {
		return "", err
	}
	
	if input == "" {
		return defaultValue, nil
	}
	
	index, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid choice: %w", err)
	}
	
	if index < 1 || index > len(choices) {
		return "", fmt.Errorf("choice must be between 1 and %d", len(choices))
	}
	
	return choices[index-1], nil
}

// ReadPassword reads a password from input (hidden)
func (ip *InputProvider) ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	
	// This is a simplified implementation
	// In a real implementation, you'd use a library like golang.org/x/term
	// to hide the input
	input, err := ip.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	
	return strings.TrimSpace(input), nil
}

// ReadMultiLine reads multiple lines from input
func (ip *InputProvider) ReadMultiLine(prompt string, terminator string) (string, error) {
	fmt.Println(prompt)
	fmt.Printf("Enter lines (terminate with '%s'):\n", terminator)
	
	var lines []string
	for {
		line, err := ip.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		
		line = strings.TrimSpace(line)
		if line == terminator {
			break
		}
		
		lines = append(lines, line)
	}
	
	return strings.Join(lines, "\n"), nil
}

// Confirm asks for confirmation
func (ip *InputProvider) Confirm(message string) (bool, error) {
	return ip.ReadBool(fmt.Sprintf("%s (y/N): ", message))
}

// ConfirmWithDefault asks for confirmation with a default value
func (ip *InputProvider) ConfirmWithDefault(message string, defaultValue bool) (bool, error) {
	return ip.ReadBoolWithDefault(message, defaultValue)
}

// InputValidator validates input
type InputValidator struct {
	validators map[string]func(string) error
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{
		validators: make(map[string]func(string) error),
	}
}

// AddValidator adds a validator for a field
func (iv *InputValidator) AddValidator(field string, validator func(string) error) {
	iv.validators[field] = validator
}

// Validate validates input
func (iv *InputValidator) Validate(field, value string) error {
	validator, exists := iv.validators[field]
	if !exists {
		return nil
	}
	return validator(value)
}

// Common validators
var (
	// Required validator
	Required = func(field string) func(string) error {
		return func(value string) error {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s is required", field)
			}
			return nil
		}
	}
	
	// Email validator
	Email = func(value string) error {
		if value == "" {
			return nil
		}
		// Simple email validation
		if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
			return fmt.Errorf("invalid email format")
		}
		return nil
	}
	
	// URL validator
	URL = func(value string) error {
		if value == "" {
			return nil
		}
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("URL must start with http:// or https://")
		}
		return nil
	}
	
	// MinLength validator
	MinLength = func(min int) func(string) error {
		return func(value string) error {
			if len(value) < min {
				return fmt.Errorf("must be at least %d characters long", min)
			}
			return nil
		}
	}
	
	// MaxLength validator
	MaxLength = func(max int) func(string) error {
		return func(value string) error {
			if len(value) > max {
				return fmt.Errorf("must be at most %d characters long", max)
			}
			return nil
		}
	}
	
	// MinMaxLength validator
	MinMaxLength = func(min, max int) func(string) error {
		return func(value string) error {
			if len(value) < min {
				return fmt.Errorf("must be at least %d characters long", min)
			}
			if len(value) > max {
				return fmt.Errorf("must be at most %d characters long", max)
			}
			return nil
		}
	}
)

// InputHelper provides helper functions for input
type InputHelper struct {
	provider  *InputProvider
	validator *InputValidator
}

// NewInputHelper creates a new input helper
func NewInputHelper() *InputHelper {
	return &InputHelper{
		provider:  NewInputProvider(),
		validator: NewInputValidator(),
	}
}

// ReadValidatedString reads a string with validation
func (ih *InputHelper) ReadValidatedString(field, prompt string, validators ...func(string) error) (string, error) {
	for {
		value, err := ih.provider.ReadString(prompt)
		if err != nil {
			return "", err
		}
		
		// Validate
		valid := true
		for _, validator := range validators {
			if err := validator(value); err != nil {
				fmt.Printf("Error: %v\n", err)
				valid = false
				break
			}
		}
		
		if valid {
			return value, nil
		}
	}
}

// ReadValidatedStringWithDefault reads a string with validation and default
func (ih *InputHelper) ReadValidatedStringWithDefault(field, prompt, defaultValue string, validators ...func(string) error) (string, error) {
	for {
		value, err := ih.provider.ReadStringWithDefault(prompt, defaultValue)
		if err != nil {
			return "", err
		}
		
		// Validate
		valid := true
		for _, validator := range validators {
			if err := validator(value); err != nil {
				fmt.Printf("Error: %v\n", err)
				valid = false
				break
			}
		}
		
		if valid {
			return value, nil
		}
	}
}

// ReadValidatedInt reads an integer with validation
func (ih *InputHelper) ReadValidatedInt(field, prompt string, validators ...func(int) error) (int, error) {
	for {
		value, err := ih.provider.ReadInt(prompt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		// Validate
		valid := true
		for _, validator := range validators {
			if err := validator(value); err != nil {
				fmt.Printf("Error: %v\n", err)
				valid = false
				break
			}
		}
		
		if valid {
			return value, nil
		}
	}
}

// ReadValidatedIntWithDefault reads an integer with validation and default
func (ih *InputHelper) ReadValidatedIntWithDefault(field, prompt string, defaultValue int, validators ...func(int) error) (int, error) {
	for {
		value, err := ih.provider.ReadIntWithDefault(prompt, defaultValue)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		// Validate
		valid := true
		for _, validator := range validators {
			if err := validator(value); err != nil {
				fmt.Printf("Error: %v\n", err)
				valid = false
				break
			}
		}
		
		if valid {
			return value, nil
		}
	}
}
