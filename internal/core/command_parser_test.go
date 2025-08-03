package core

import (
	"reflect"
	"testing"
)

func TestCommandParser(t *testing.T) {
	parser := NewCommandParser()
	
	tests := []struct {
		name     string
		input    string
		expected *ParsedCommand
		wantErr  bool
	}{
		{
			name:  "simple command",
			input: "help",
			expected: &ParsedCommand{
				Path:     []string{"help"},
				Args:     []string{},
				Flags:    map[string]string{},
				RawInput: "help",
			},
		},
		{
			name:  "slash command",
			input: "stream/tap",
			expected: &ParsedCommand{
				Path:     []string{"stream", "tap"},
				Args:     []string{},
				Flags:    map[string]string{},
				RawInput: "stream/tap",
			},
		},
		{
			name:  "command with flags",
			input: "stream/tap --auto-discover --duration 30s",
			expected: &ParsedCommand{
				Path:     []string{"stream", "tap"},
				Args:     []string{},
				Flags:    map[string]string{"auto-discover": "true", "duration": "30s"},
				RawInput: "stream/tap --auto-discover --duration 30s",
			},
		},
		{
			name:  "command with short flags",
			input: "stream/tap -a -d 30s",
			expected: &ParsedCommand{
				Path:     []string{"stream", "tap"},
				Args:     []string{},
				Flags:    map[string]string{"a": "true", "d": "30s"},
				RawInput: "stream/tap -a -d 30s",
			},
		},
		{
			name:  "command with quoted argument",
			input: `stream/tap --output "file:/tmp/my capture.jsonl"`,
			expected: &ParsedCommand{
				Path:     []string{"stream", "tap"},
				Args:     []string{},
				Flags:    map[string]string{"output": "file:/tmp/my capture.jsonl"},
				RawInput: `stream/tap --output "file:/tmp/my capture.jsonl"`,
			},
		},
		{
			name:  "command with positional args",
			input: "help stream/tap",
			expected: &ParsedCommand{
				Path:     []string{"help"},
				Args:     []string{"stream/tap"},
				Flags:    map[string]string{},
				RawInput: "help stream/tap",
			},
		},
		{
			name:  "complex command",
			input: `stream/tap --pid 12345 --duration 5m --output file:"/var/log/capture.jsonl" --filter 'contains("password")'`,
			expected: &ParsedCommand{
				Path:     []string{"stream", "tap"},
				Args:     []string{},
				Flags:    map[string]string{
					"pid":      "12345",
					"duration": "5m",
					"output":   "file:/var/log/capture.jsonl",
					"filter":   `contains("password")`,
				},
				RawInput: `stream/tap --pid 12345 --duration 5m --output file:"/var/log/capture.jsonl" --filter 'contains("password")'`,
			},
		},
		{
			name:  "flag with equals syntax",
			input: "stream/tap --duration=30s --output=stdout",
			expected: &ParsedCommand{
				Path:     []string{"stream", "tap"},
				Args:     []string{},
				Flags:    map[string]string{"duration": "30s", "output": "stdout"},
				RawInput: "stream/tap --duration=30s --output=stdout",
			},
		},
		{
			name:  "combined short flags",
			input: "test -abc",
			expected: &ParsedCommand{
				Path:     []string{"test"},
				Args:     []string{},
				Flags:    map[string]string{"a": "true", "b": "true", "c": "true"},
				RawInput: "test -abc",
			},
		},
		{
			name:    "unclosed quote",
			input:   `test "unclosed`,
			wantErr: true,
		},
		{
			name:  "escaped quotes",
			input: `test "say \"hello\" world"`,
			expected: &ParsedCommand{
				Path:     []string{"test"},
				Args:     []string{`say "hello" world`},
				Flags:    map[string]string{},
				RawInput: `test "say \"hello\" world"`,
			},
		},
		{
			name:  "multi-level slash command",
			input: "integrations/prometheus/enable --port 9100",
			expected: &ParsedCommand{
				Path:     []string{"integrations", "prometheus", "enable"},
				Args:     []string{},
				Flags:    map[string]string{"port": "9100"},
				RawInput: "integrations/prometheus/enable --port 9100",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			
			if !reflect.DeepEqual(got.Path, tt.expected.Path) {
				t.Errorf("Parse() Path = %v, want %v", got.Path, tt.expected.Path)
			}
			if !reflect.DeepEqual(got.Args, tt.expected.Args) {
				t.Errorf("Parse() Args = %v, want %v", got.Args, tt.expected.Args)
			}
			if !reflect.DeepEqual(got.Flags, tt.expected.Flags) {
				t.Errorf("Parse() Flags = %v, want %v", got.Flags, tt.expected.Flags)
			}
			if got.RawInput != tt.expected.RawInput {
				t.Errorf("Parse() RawInput = %v, want %v", got.RawInput, tt.expected.RawInput)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	parser := NewCommandParser()
	
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "simple tokens",
			input:    "one two three",
			expected: []string{"one", "two", "three"},
		},
		{
			name:     "quoted string",
			input:    `one "two three" four`,
			expected: []string{"one", "two three", "four"},
		},
		{
			name:     "single quotes",
			input:    `one 'two three' four`,
			expected: []string{"one", "two three", "four"},
		},
		{
			name:     "escaped quotes",
			input:    `say "hello \"world\""`,
			expected: []string{"say", `hello "world"`},
		},
		{
			name:     "escaped backslash",
			input:    `path "C:\\Users\\test"`,
			expected: []string{"path", `C:\Users\test`},
		},
		{
			name:     "mixed quotes",
			input:    `test "double" 'single'`,
			expected: []string{"test", "double", "single"},
		},
		{
			name:    "unclosed quote",
			input:   `test "unclosed`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.tokenize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("tokenize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("tokenize() = %v, want %v", got, tt.expected)
			}
		})
	}
}