package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PromptString asks for a string value with a default fallback.
func PromptString(reader *bufio.Reader, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return defaultValue, nil
	}
	return trimmed, nil
}

// PromptInt asks for an integer value with a default fallback.
func PromptInt(reader *bufio.Reader, label string, defaultValue int) (int, error) {
	fmt.Printf("%s [%d]: ", label, defaultValue)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return defaultValue, err
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(trimmed)
	if err != nil {
		return defaultValue, fmt.Errorf("invalid integer '%s': %w", trimmed, err)
	}
	return v, nil
}

// PromptIntList asks for a list of integers (comma- or space-separated).
func PromptIntList(reader *bufio.Reader, label string) ([]int, error) {
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil, nil
	}

	// Replace commas with spaces and split fields
	cleaned := strings.ReplaceAll(trimmed, ",", " ")
	parts := strings.Fields(cleaned)
	var result []int
	for _, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid integer '%s': %w", p, err)
		}
		result = append(result, v)
	}
	return result, nil
}

// PromptBool asks for a boolean value (y/n, yes/no, true/false) with a default fallback.
func PromptBool(reader *bufio.Reader, label string, defaultValue bool) (bool, error) {
	defStr := "n"
	if defaultValue {
		defStr = "y"
	}
	fmt.Printf("%s (y/n) [%s]: ", label, defStr)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return defaultValue, err
	}
	trimmed := strings.ToLower(strings.TrimSpace(line))
	if trimmed == "" {
		return defaultValue, nil
	}
	if trimmed == "y" || trimmed == "yes" || trimmed == "true" || trimmed == "1" {
		return true, nil
	}
	if trimmed == "n" || trimmed == "no" || trimmed == "false" || trimmed == "0" {
		return false, nil
	}
	return defaultValue, nil
}
