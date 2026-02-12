package render

import (
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	r := New()

	tests := []struct {
		text    string
		percent int
		color   string
	}{
		{"ok", 0, Green},
		{"ok", 49, Green},
		{"warn", 50, Yellow},
		{"warn", 79, Yellow},
		{"crit", 80, Red},
		{"crit", 100, Red},
	}

	for _, tt := range tests {
		got := r.Colorize(tt.text, tt.percent)
		if !strings.HasPrefix(got, tt.color) {
			t.Errorf("Colorize(%q, %d) should start with color %q, got %q", tt.text, tt.percent, tt.color, got)
		}
		if !strings.HasSuffix(got, Reset) {
			t.Errorf("Colorize(%q, %d) should end with Reset", tt.text, tt.percent)
		}
	}
}

func TestColorForPercent(t *testing.T) {
	tests := []struct {
		percent int
		want    string
	}{
		{0, Green},
		{49, Green},
		{50, Yellow},
		{79, Yellow},
		{80, Red},
		{100, Red},
	}
	for _, tt := range tests {
		got := ColorForPercent(tt.percent)
		if got != tt.want {
			t.Errorf("ColorForPercent(%d) = %q, want %q", tt.percent, got, tt.want)
		}
	}
}

func TestMakeBar(t *testing.T) {
	r := New()

	tests := []struct {
		percent int
		width   int
		filled  int
		empty   int
	}{
		{0, 8, 0, 8},
		{25, 8, 2, 6},
		{50, 8, 4, 4},
		{80, 8, 6, 2},
		{100, 8, 8, 0},
		{-5, 8, 0, 8},
		{150, 8, 8, 0},
	}

	for _, tt := range tests {
		got := r.MakeBar(tt.percent, tt.width)
		filledCount := strings.Count(got, "█")
		emptyCount := strings.Count(got, "░")
		if filledCount != tt.filled {
			t.Errorf("MakeBar(%d, %d): filled=%d, want %d", tt.percent, tt.width, filledCount, tt.filled)
		}
		if emptyCount != tt.empty {
			t.Errorf("MakeBar(%d, %d): empty=%d, want %d", tt.percent, tt.width, emptyCount, tt.empty)
		}
	}
}

func TestFormatTokens(t *testing.T) {
	r := New()

	// FormatTokens: 0 decimals (for context max)
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1K"},
		{1500, "2K"},
		{32768, "33K"},
		{100000, "100K"},
		{200000, "200K"},
	}

	for _, tt := range tests {
		got := r.FormatTokens(tt.input)
		if got != tt.expected {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatTokensF(t *testing.T) {
	r := New()

	// FormatTokensF: 1 decimal (for used tokens, burn rate)
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{10000, "10.0K"},
		{12300, "12.3K"},
		{100000, "100.0K"},
		{200000, "200.0K"},
	}

	for _, tt := range tests {
		got := r.FormatTokensF(tt.input)
		if got != tt.expected {
			t.Errorf("FormatTokensF(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatCost(t *testing.T) {
	r := New()

	tests := []struct {
		input    float64
		expected string
	}{
		{0, "$0.00"},
		{0.5, "$0.50"},
		{1.234, "$1.23"},
		{20.0, "$20.00"},
	}

	for _, tt := range tests {
		got := r.FormatCost(tt.input)
		if got != tt.expected {
			t.Errorf("FormatCost(%f) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	r := New()

	tests := []struct {
		ms       int
		expected string
	}{
		{0, "0m"},
		{60000, "1m"},
		{300000, "5m"},
		{3600000, "60m"},
	}

	for _, tt := range tests {
		got := r.FormatDuration(tt.ms)
		if got != tt.expected {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.ms, got, tt.expected)
		}
	}
}

func TestDim(t *testing.T) {
	r := New()
	got := r.Dim("test")
	if got != DimCode+"test"+Reset {
		t.Errorf("Dim(%q) = %q, want %q", "test", got, DimCode+"test"+Reset)
	}
}

func TestColor(t *testing.T) {
	r := New()

	tests := []struct {
		text  string
		color string
		want  string
	}{
		{"ok", Green, Green + "ok" + Reset},
		{"warn", Yellow, Yellow + "warn" + Reset},
		{"crit", Red, Red + "crit" + Reset},
		{"info", Cyan, Cyan + "info" + Reset},
		{"tpm", Magenta, Magenta + "tpm" + Reset},
	}

	for _, tt := range tests {
		got := r.Color(tt.text, tt.color)
		if got != tt.want {
			t.Errorf("Color(%q, %q) = %q, want %q", tt.text, tt.color, got, tt.want)
		}
	}
}
