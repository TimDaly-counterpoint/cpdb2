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
	reader := bufio.NewReader(strings.NewReader("1, 2, 3\n1-3, 5-7\n1, 3-5, 8\n\n"))

	list, err := PromptIntList(reader, "Enter numbers")
	if err != nil || len(list) != 3 || list[0] != 1 || list[1] != 2 || list[2] != 3 {
		t.Errorf("got %v, want [1 2 3], err: %v", list, err)
	}

	list, err = PromptIntList(reader, "Enter numbers")
	if err != nil || len(list) != 6 || list[0] != 1 || list[2] != 3 || list[3] != 5 || list[5] != 7 {
		t.Errorf("got %v, want [1 2 3 5 6 7], err: %v", list, err)
	}

	list, err = PromptIntList(reader, "Enter numbers")
	if err != nil || len(list) != 5 || list[0] != 1 || list[1] != 3 || list[2] != 4 || list[3] != 5 || list[4] != 8 {
		t.Errorf("got %v, want [1 3 4 5 8], err: %v", list, err)
	}

	list, err = PromptIntList(reader, "Enter numbers")
	if err != nil || len(list) != 0 {
		t.Errorf("got %v, want [], err: %v", list, err)
	}
}

func TestParseIntList(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
		wantErr  bool
	}{
		{"", nil, false},
		{"   ", nil, false},
		{"1", []int{1}, false},
		{"1, 2, 3", []int{1, 2, 3}, false},
		{"1,2,3", []int{1, 2, 3}, false},
		{"1 2 3", []int{1, 2, 3}, false},
		{"1-5", []int{1, 2, 3, 4, 5}, false},
		{"1 - 5", []int{1, 2, 3, 4, 5}, false},
		{"1-3, 5-7", []int{1, 2, 3, 5, 6, 7}, false},
		{"1-3,5-7", []int{1, 2, 3, 5, 6, 7}, false},
		{"1, 3-5, 8", []int{1, 3, 4, 5, 8}, false},
		{"1 3-5 8", []int{1, 3, 4, 5, 8}, false},
		{"3-3", []int{3}, false},
		{"5-3", []int{5, 4, 3}, false},
		{", 1, 2-4 , 7 , ", []int{1, 2, 3, 4, 7}, false},
		{"1-2-3", nil, true},
		{"1-", nil, true},
		{"-", nil, true},
		{"abc", nil, true},
		{"1, abc, 3", nil, true},
		{"1-foo", nil, true},
	}

	for _, tc := range tests {
		got, err := ParseIntList(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseIntList(%q) err = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr {
			if len(got) != len(tc.expected) {
				t.Errorf("ParseIntList(%q) = %v, want %v", tc.input, got, tc.expected)
				continue
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("ParseIntList(%q)[%d] = %d, want %d", tc.input, i, got[i], tc.expected[i])
				}
			}
		}
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
