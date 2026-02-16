package context

import (
	"strings"
	"testing"

	"github.com/Benniphx/claude-statusline/adapter/render"
	"github.com/Benniphx/claude-statusline/core/types"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name        string
		input       types.Input
		model       types.ModelInfo
		wantPercent int
		wantTokens  int
		wantInitial bool
	}{
		{
			name: "with used_percentage from API",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 200000,
					UsedPercentage:    ptrFloat(45.5),
					CurrentUsage: types.CurrentUsage{
						InputTokens: 50000,
					},
				},
				Cost: types.Cost{TotalDurationMS: 60000},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 45,
			wantTokens:  50000,
		},
		{
			name: "calculated percentage",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 200000,
					CurrentUsage: types.CurrentUsage{
						InputTokens:              80000,
						CacheCreationInputTokens: 10000,
						CacheReadInputTokens:     10000,
					},
				},
				Cost: types.Cost{TotalDurationMS: 60000},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 50,
			wantTokens:  100000,
		},
		{
			name: "legacy fallback (no current_usage)",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 200000,
					TotalInputTokens:  60000,
					TotalOutputTokens: 40000,
				},
				Cost: types.Cost{TotalDurationMS: 60000},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 50,
			wantTokens:  100000,
		},
		{
			name: "capped at context size",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 100000,
					CurrentUsage: types.CurrentUsage{
						InputTokens: 150000,
					},
				},
				Cost: types.Cost{TotalDurationMS: 60000},
			},
			model:       types.ModelInfo{DefaultContext: 100000},
			wantPercent: 100,
			wantTokens:  100000,
		},
		{
			name: "initial state",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 200000,
				},
				Cost: types.Cost{TotalDurationMS: 0},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 0,
			wantInitial: true,
		},
		{
			name: "stale percentage after clear (tokens=0)",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 200000,
					UsedPercentage:    ptrFloat(95.0),
				},
				Cost: types.Cost{TotalDurationMS: 0},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 0,
			wantInitial: true,
		},
		{
			name: "stale percentage mismatch (API=95%, tokens=2%)",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					ContextWindowSize: 200000,
					UsedPercentage:    ptrFloat(95.0),
					CurrentUsage: types.CurrentUsage{
						InputTokens: 4000,
					},
				},
				Cost: types.Cost{TotalDurationMS: 60000},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 2,
			wantTokens:  4000,
		},
		{
			name: "uses model default when API gives 0",
			input: types.Input{
				ContextWindow: types.ContextWindow{
					CurrentUsage: types.CurrentUsage{
						InputTokens: 50000,
					},
				},
				Cost: types.Cost{TotalDurationMS: 60000},
			},
			model:       types.ModelInfo{DefaultContext: 200000},
			wantPercent: 25,
			wantTokens:  50000,
		},
	}

	cfg := types.DefaultConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			display := Calculate(tt.input, tt.model, cfg)
			if display.PercentUsed != tt.wantPercent {
				t.Errorf("PercentUsed = %d, want %d", display.PercentUsed, tt.wantPercent)
			}
			if tt.wantTokens > 0 && display.TokensUsed != tt.wantTokens {
				t.Errorf("TokensUsed = %d, want %d", display.TokensUsed, tt.wantTokens)
			}
			if display.IsInitial != tt.wantInitial {
				t.Errorf("IsInitial = %v, want %v", display.IsInitial, tt.wantInitial)
			}
		})
	}
}

func TestRender(t *testing.T) {
	r := render.New()
	cfg := types.DefaultConfig()

	// Normal render
	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: types.CurrentUsage{
				InputTokens: 90000,
			},
		},
		Cost: types.Cost{TotalDurationMS: 300000},
	}
	model := types.ModelInfo{DefaultContext: 200000}
	display := Calculate(input, model, cfg)
	result := Render(display, cfg, r)

	if !strings.Contains(result, "Ctx:") {
		t.Error("should contain 'Ctx:'")
	}
	if !strings.Contains(result, "45%") {
		t.Error("should contain '45%'")
	}
	if !strings.Contains(result, "90.0K") {
		t.Errorf("should contain '90.0K' (1 decimal), got: %s", result)
	}
	if !strings.Contains(result, "200K") {
		t.Errorf("should contain '200K' (0 decimal), got: %s", result)
	}
	// Bar should be 8 chars
	if strings.Count(result, "█")+strings.Count(result, "░") != 8 {
		t.Error("bar should be 8 characters")
	}

	// Initial state: should show bar + "--" + (0/200K)
	input2 := types.Input{
		ContextWindow: types.ContextWindow{ContextWindowSize: 200000},
	}
	display2 := Calculate(input2, model, cfg)
	result2 := Render(display2, cfg, r)
	if !strings.Contains(result2, "--") {
		t.Error("initial state should show '--'")
	}
	if !strings.Contains(result2, "0/200K") {
		t.Errorf("initial state should show '(0/200K)', got: %s", result2)
	}
	if strings.Count(result2, "░") != 8 {
		t.Error("initial state bar should be 8 empty chars")
	}
}

func TestRenderWithWarning(t *testing.T) {
	r := render.New()
	cfg := types.DefaultConfig()
	cfg.ContextWarningThreshold = 40

	input := types.Input{
		ContextWindow: types.ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: types.CurrentUsage{
				InputTokens: 90000,
			},
		},
		Cost: types.Cost{TotalDurationMS: 60000},
	}
	model := types.ModelInfo{DefaultContext: 200000}
	display := Calculate(input, model, cfg)
	result := Render(display, cfg, r)

	if !strings.Contains(result, "⚠️") {
		t.Error("should contain warning when over threshold")
	}
}

func ptrFloat(f float64) *float64 {
	return &f
}
