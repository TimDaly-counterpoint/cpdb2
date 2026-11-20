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

// ParseIntList parses a string containing a list of integers, ranges (e.g. "1-5"), or a combination,
// separated by commas or spaces.
func ParseIntList(input string) ([]int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	var sb strings.Builder
	runes := []rune(trimmed)
	n := len(runes)
	for i := 0; i < n; i++ {
		r := runes[i]
		if r == ',' {
			sb.WriteRune(' ')
		} else if r == '-' {
			s := sb.String()
			trimmedEnd := strings.TrimRight(s, " \t")
			if len(trimmedEnd) > 0 {
				sb.Reset()
				sb.WriteString(trimmedEnd)
			}
			sb.WriteRune('-')
			for i+1 < n && (runes[i+1] == ' ' || runes[i+1] == '\t') {
				i++
			}
		} else {
			sb.WriteRune(r)
		}
	}

	parts := strings.Fields(sb.String())
	var result []int

	for _, p := range parts {
		if strings.Contains(p, "-") {
			dashIdx := strings.Index(p, "-")
			if dashIdx == 0 && !strings.Contains(p[1:], "-") {
				v, err := strconv.Atoi(p)
				if err != nil {
					return nil, fmt.Errorf("invalid integer '%s': %w", p, err)
				}
				result = append(result, v)
				continue
			}

			subParts := strings.Split(p, "-")
			if len(subParts) == 2 && subParts[0] != "" && subParts[1] != "" {
				start, err1 := strconv.Atoi(subParts[0])
				end, err2 := strconv.Atoi(subParts[1])
				if err1 != nil || err2 != nil {
					return nil, fmt.Errorf("invalid integer range '%s'", p)
				}
				if start <= end {
					for i := start; i <= end; i++ {
						result = append(result, i)
					}
				} else {
					for i := start; i >= end; i-- {
						result = append(result, i)
					}
				}
				continue
			}

			v, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid integer or range '%s'", p)
			}
			result = append(result, v)
		} else {
			v, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid integer '%s': %w", p, err)
			}
			result = append(result, v)
		}
	}
	return result, nil
}

// PromptIntList asks for a list of integers or ranges (comma- or space-separated).
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
	return ParseIntList(trimmed)
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
