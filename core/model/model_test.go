package model

import (
	"testing"

	"github.com/Benniphx/claude-statusline/core/types"
)

// mockOllama implements ports.OllamaClient for testing.
type mockOllama struct {
	contextSize int
	err         error
}

func (m *mockOllama) GetContextSize(model string) (int, error) {
	return m.contextSize, m.err
}

func TestResolveClaudeModels(t *testing.T) {
	cfg := types.DefaultConfig()
	tests := []struct {
		modelID     string
		displayName string
		wantShort   string
		wantContext int
		wantLocal   bool
	}{
		{"claude-opus-4-5-20251101", "Claude Opus 4.5", "Opus 4.5", 200000, false},
		{"some-opus-4.5-variant", "Custom", "Opus 4.5", 200000, false},
		{"claude-opus-4-6-20260101", "Claude Opus 4.6", "Opus 4.6", 200000, false},
		{"claude-sonnet-4-6-20260101", "Claude Sonnet 4.6", "Sonnet 4.6", 200000, false},
		{"some-sonnet-4.6-variant", "Custom", "Sonnet 4.6", 200000, false},
		{"claude-sonnet-4-5-20250929", "Claude Sonnet 4.5", "Sonnet 4.5", 200000, false},
		{"claude-sonnet-4-20250514", "Claude Sonnet 4", "Sonnet 4", 200000, false},
		{"some-sonnet-4-2025-variant", "Custom", "Sonnet 4", 200000, false},
		{"claude-3-5-sonnet-20241022", "Claude Sonnet 3.5", "Sonnet 3.5", 200000, false},
		{"some-sonnet-3.5-variant", "Custom", "Sonnet 3.5", 200000, false},
		{"claude-3-opus-20240229", "Claude Opus 3", "Opus 3", 200000, false},
		{"claude-3-5-haiku-20241022", "Claude Haiku 3.5", "Haiku 3.5", 200000, false},
		{"some-haiku-3-variant", "Custom", "Haiku 3", 200000, false},
		{"claude-opus-4-20260301", "Claude Opus 4", "Opus 4", 200000, false},
		{"some-opus-4-variant", "Custom", "Opus 4", 200000, false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := Resolve(tt.modelID, tt.displayName, nil, "", cfg)
			if info.ShortName != tt.wantShort {
				t.Errorf("ShortName = %q, want %q", info.ShortName, tt.wantShort)
			}
			if info.DefaultContext != tt.wantContext {
				t.Errorf("DefaultContext = %d, want %d", info.DefaultContext, tt.wantContext)
			}
			if info.IsLocal != tt.wantLocal {
				t.Errorf("IsLocal = %v, want %v", info.IsLocal, tt.wantLocal)
			}
		})
	}
}

func TestResolveOllamaModels(t *testing.T) {
	cfg := types.DefaultConfig()
	tests := []struct {
		modelID   string
		wantShort string
		wantLocal bool
	}{
		{"ollama:qwen3-coder-latest", "🦙 Qwen3", true},
		{"ollama:qwen2.5-coder-7b", "🦙 Qwen2.5", true},
		{"ollama:llama3-8b", "🦙 Llama3", true},
		{"ollama:llama2-7b", "🦙 Llama2", true},
		{"ollama:codellama-7b", "🦙 CodeLlama", true},
		{"ollama:mistral-7b", "🦙 Mistral", true},
		{"ollama:deepseek-coder", "🦙 DeepSeek", true},
		{"ollama:phi-3-mini", "🦙 Phi", true},
		{"ollama/llama3-8b", "🦙 Llama3", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := Resolve(tt.modelID, "", nil, "", cfg)
			if info.ShortName != tt.wantShort {
				t.Errorf("ShortName = %q, want %q", info.ShortName, tt.wantShort)
			}
			if info.IsLocal != tt.wantLocal {
				t.Errorf("IsLocal = %v, want %v", info.IsLocal, tt.wantLocal)
			}
		})
	}
}

func TestResolveOllamaWithContextSize(t *testing.T) {
	cfg := types.DefaultConfig()
	mock := &mockOllama{contextSize: 131072}
	info := Resolve("ollama:llama3-8b", "", mock, "/tmp", cfg)

	if info.DefaultContext != 131072 {
		t.Errorf("DefaultContext = %d, want 131072", info.DefaultContext)
	}
}

func TestResolveLocalModel(t *testing.T) {
	cfg := types.DefaultConfig()
	info := Resolve("some-local-model", "", nil, "", cfg)
	if info.ShortName != "🦙 Local" {
		t.Errorf("ShortName = %q, want %q", info.ShortName, "🦙 Local")
	}
	if !info.IsLocal {
		t.Error("IsLocal should be true")
	}
}

func TestResolveFallback(t *testing.T) {
	cfg := types.DefaultConfig()
	info := Resolve("unknown-model-id", "Claude Something", nil, "", cfg)
	if info.ShortName != "Something" {
		t.Errorf("ShortName = %q, want %q", info.ShortName, "Something")
	}

	info2 := Resolve("unknown-model-id", "", nil, "", cfg)
	if info2.ShortName != "unknown-model-id" {
		t.Errorf("ShortName = %q, want %q", info2.ShortName, "unknown-model-id")
	}
}

func TestCostWeight(t *testing.T) {
	cfg := types.DefaultConfig()
	tests := []struct {
		modelID     string
		displayName string
		wantWeight  float64
	}{
		{"claude-opus-4-6-20260101", "Claude Opus 4.6", 5.0},
		{"claude-opus-4-5-20251101", "Claude Opus 4.5", 5.0},
		{"claude-sonnet-4-20250514", "Claude Sonnet 4", 1.0},
		{"claude-sonnet-4-5-20250929", "Claude Sonnet 4.5", 1.0},
		{"claude-3-5-haiku-20241022", "Claude Haiku 3.5", 0.25},
		{"claude-haiku-4-20260101", "Claude Haiku 4", 0.25},
		{"claude-opus-4-20260301", "Claude Opus 4", 5.0},
		{"unknown-model", "Claude Something", 1.0}, // fallback
		{"unknown-model", "", 1.0},                  // unknown fallback
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := Resolve(tt.modelID, tt.displayName, nil, "", cfg)
			if info.CostWeight != tt.wantWeight {
				t.Errorf("CostWeight = %f, want %f", info.CostWeight, tt.wantWeight)
			}
		})
	}
}

func TestCostWeightFromConfig(t *testing.T) {
	cfg := types.DefaultConfig()

	if w := CostWeight(FamilyHaiku, cfg); w != 0.25 {
		t.Errorf("Haiku weight = %f, want 0.25", w)
	}
	if w := CostWeight(FamilySonnet, cfg); w != 1.0 {
		t.Errorf("Sonnet weight = %f, want 1.0", w)
	}
	if w := CostWeight(FamilyOpus, cfg); w != 5.0 {
		t.Errorf("Opus weight = %f, want 5.0", w)
	}
	if w := CostWeight(FamilyUnknown, cfg); w != 1.0 {
		t.Errorf("Unknown weight = %f, want 1.0", w)
	}

	// Custom weights
	cfg.CostWeightOpus = 10.0
	if w := CostWeight(FamilyOpus, cfg); w != 10.0 {
		t.Errorf("Custom Opus weight = %f, want 10.0", w)
	}
}

func TestDetectFamilyFromDisplayName(t *testing.T) {
	cfg := types.DefaultConfig()

	// Fallback: display name contains family hint
	info := Resolve("custom-unknown-id", "Claude Opus Custom", nil, "", cfg)
	if info.CostWeight != 5.0 {
		t.Errorf("CostWeight from display name 'Opus' = %f, want 5.0", info.CostWeight)
	}

	info2 := Resolve("custom-unknown-id", "Claude Haiku Custom", nil, "", cfg)
	if info2.CostWeight != 0.25 {
		t.Errorf("CostWeight from display name 'Haiku' = %f, want 0.25", info2.CostWeight)
	}
}

func TestResolveUsesConfigWeights(t *testing.T) {
	cfg := types.DefaultConfig()
	cfg.CostWeightOpus = 10.0
	cfg.CostWeightHaiku = 0.5

	// Opus model should pick up custom config weight
	info := Resolve("claude-opus-4-6-20260101", "Claude Opus 4.6", nil, "", cfg)
	if info.CostWeight != 10.0 {
		t.Errorf("Opus CostWeight with custom config = %f, want 10.0", info.CostWeight)
	}

	// Haiku model should pick up custom config weight
	info2 := Resolve("claude-3-5-haiku-20241022", "Claude Haiku 3.5", nil, "", cfg)
	if info2.CostWeight != 0.5 {
		t.Errorf("Haiku CostWeight with custom config = %f, want 0.5", info2.CostWeight)
	}

	// Sonnet should still use default
	info3 := Resolve("claude-sonnet-4-20250514", "Claude Sonnet 4", nil, "", cfg)
	if info3.CostWeight != 1.0 {
		t.Errorf("Sonnet CostWeight with default config = %f, want 1.0", info3.CostWeight)
	}
}
