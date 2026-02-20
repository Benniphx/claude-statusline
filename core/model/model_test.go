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
		{"claude-sonnet-4-5-20250929", "Claude Sonnet 4.5", "Sonnet 4.5", 200000, false},
		{"claude-sonnet-4-20250514", "Claude Sonnet 4", "Sonnet 4", 200000, false},
		{"some-sonnet-4-2025-variant", "Custom", "Sonnet 4", 200000, false},
		{"claude-3-5-sonnet-20241022", "Claude Sonnet 3.5", "Sonnet 3.5", 200000, false},
		{"some-sonnet-3.5-variant", "Custom", "Sonnet 3.5", 200000, false},
		{"claude-3-opus-20240229", "Claude Opus 3", "Opus 3", 200000, false},
		{"claude-3-5-haiku-20241022", "Claude Haiku 3.5", "Haiku 3.5", 200000, false},
		{"some-haiku-3-variant", "Custom", "Haiku 3", 200000, false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			info := Resolve(tt.modelID, tt.displayName, nil, "")
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
			info := Resolve(tt.modelID, "", nil, "")
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
	mock := &mockOllama{contextSize: 131072}
	info := Resolve("ollama:llama3-8b", "", mock, "/tmp")

	if info.DefaultContext != 131072 {
		t.Errorf("DefaultContext = %d, want 131072", info.DefaultContext)
	}
}

func TestResolveLocalModel(t *testing.T) {
	info := Resolve("some-local-model", "", nil, "")
	if info.ShortName != "🦙 Local" {
		t.Errorf("ShortName = %q, want %q", info.ShortName, "🦙 Local")
	}
	if !info.IsLocal {
		t.Error("IsLocal should be true")
	}
}

func TestResolveFallback(t *testing.T) {
	info := Resolve("unknown-model-id", "Claude Something", nil, "")
	if info.ShortName != "Something" {
		t.Errorf("ShortName = %q, want %q", info.ShortName, "Something")
	}

	info2 := Resolve("unknown-model-id", "", nil, "")
	if info2.ShortName != "unknown-model-id" {
		t.Errorf("ShortName = %q, want %q", info2.ShortName, "unknown-model-id")
	}
}

func TestResolveCostTier(t *testing.T) {
	tests := []struct {
		modelID     string
		displayName string
		wantTier    types.CostTier
	}{
		// Opus models → CostTierOpus
		{"claude-opus-4-5-20251101", "Claude Opus 4.5", types.CostTierOpus},
		{"claude-opus-4-6-20260101", "Claude Opus 4.6", types.CostTierOpus},
		{"claude-3-opus-20240229", "Claude Opus 3", types.CostTierOpus},
		// Sonnet models → CostTierSonnet
		{"claude-sonnet-4-5-20250929", "Claude Sonnet 4.5", types.CostTierSonnet},
		{"claude-sonnet-4-20250514", "Claude Sonnet 4", types.CostTierSonnet},
		{"claude-3-5-sonnet-20241022", "Claude Sonnet 3.5", types.CostTierSonnet},
		// Haiku models → CostTierHaiku
		{"claude-3-5-haiku-20241022", "Claude Haiku 3.5", types.CostTierHaiku},
		{"some-haiku-3-variant", "Custom", types.CostTierHaiku},
		{"claude-haiku-4-variant", "Haiku 4", types.CostTierHaiku},
		// Fallback from display_name
		{"unknown-id", "Claude Opus 4.6", types.CostTierOpus},
		{"unknown-id", "Claude Haiku 3.5", types.CostTierHaiku},
		{"unknown-id", "Claude Sonnet 4", types.CostTierSonnet},
		// Unknown → default Sonnet
		{"unknown-id", "Unknown Model", types.CostTierSonnet},
		{"unknown-id", "", types.CostTierSonnet},
	}

	for _, tt := range tests {
		t.Run(tt.modelID+"_"+tt.displayName, func(t *testing.T) {
			info := Resolve(tt.modelID, tt.displayName, nil, "")
			if info.CostTier != tt.wantTier {
				t.Errorf("CostTier = %v, want %v", info.CostTier, tt.wantTier)
			}
		})
	}
}

func TestResolveCostTierOllama(t *testing.T) {
	// Ollama models should default to Sonnet tier
	info := Resolve("ollama:qwen3-coder-latest", "", nil, "")
	if info.CostTier != types.CostTierSonnet {
		t.Errorf("Ollama CostTier = %v, want CostTierSonnet", info.CostTier)
	}
}

func TestResolveCostTierLocal(t *testing.T) {
	// Local models should default to Sonnet tier
	info := Resolve("some-local-model", "", nil, "")
	if info.CostTier != types.CostTierSonnet {
		t.Errorf("Local CostTier = %v, want CostTierSonnet", info.CostTier)
	}
}
