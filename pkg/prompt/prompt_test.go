package prompt

import (
	"bufio"
	"strings"
	"testing"
)

func TestPromptString(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("hello\n\n"))

	val, err := PromptString(reader, "Enter text", "default")
	if err != nil || val != "hello" {
		t.Errorf("got %q, want 'hello', err: %v", val, err)
	}

	val, err = PromptString(reader, "Enter text", "default")
	if err != nil || val != "default" {
		t.Errorf("got %q, want 'default', err: %v", val, err)
	}
}

func TestPromptInt(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("42\n\n"))

	val, err := PromptInt(reader, "Enter number", 10)
	if err != nil || val != 42 {
		t.Errorf("got %d, want 42, err: %v", val, err)
	}

	val, err = PromptInt(reader, "Enter number", 10)
	if err != nil || val != 10 {
		t.Errorf("got %d, want 10, err: %v", val, err)
	}
}

func TestPromptIntList(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("1, 2, 3\n\n"))

	list, err := PromptIntList(reader, "Enter numbers")
	if err != nil || len(list) != 3 || list[0] != 1 || list[1] != 2 || list[2] != 3 {
		t.Errorf("got %v, want [1 2 3], err: %v", list, err)
	}

	list, err = PromptIntList(reader, "Enter numbers")
	if err != nil || len(list) != 0 {
		t.Errorf("got %v, want [], err: %v", list, err)
	}
}

func TestPromptBool(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("y\nn\n\n"))

	b, err := PromptBool(reader, "Confirm", false)
	if err != nil || !b {
		t.Errorf("got %v, want true, err: %v", b, err)
	}

	b, err = PromptBool(reader, "Confirm", true)
	if err != nil || b {
		t.Errorf("got %v, want false, err: %v", b, err)
	}

	b, err = PromptBool(reader, "Confirm", true)
	if err != nil || !b {
		t.Errorf("got %v, want true, err: %v", b, err)
	}
}
