package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Benniphx/claude-statusline/core/ports"
	"github.com/Benniphx/claude-statusline/core/types"
)

const defaultClaudeContext = 200000
const defaultLocalContext = 32768

// Resolve determines the short name, default context size, and locality of a model.
func Resolve(modelID, displayName string, ollama ports.OllamaClient, cacheDir string) types.ModelInfo {
	lower := strings.ToLower(modelID)

	// Check Ollama models first
	if strings.HasPrefix(lower, "ollama:") || strings.HasPrefix(lower, "ollama/") {
		return resolveOllama(modelID, ollama, cacheDir)
	}

	// Check local models
	if strings.Contains(lower, "local") || strings.Contains(lower, "localhost") {
		return types.ModelInfo{
			ShortName:      "🦙 Local",
			DefaultContext: defaultLocalContext,
			IsLocal:        true,
		}
	}

	// Claude model patterns
	if short, tier, ok := matchClaude(lower); ok {
		return types.ModelInfo{
			ShortName:      short,
			DefaultContext: defaultClaudeContext,
			IsLocal:        false,
			CostTier:       tier,
		}
	}

	// Fallback: strip "Claude " prefix from display name
	name := displayName
	if strings.HasPrefix(name, "Claude ") {
		name = strings.TrimPrefix(name, "Claude ")
	}
	if name == "" {
		name = modelID
	}

	// Detect cost tier from display name fallback
	tier := detectCostTier(displayName)

	return types.ModelInfo{
		ShortName:      name,
		DefaultContext: defaultClaudeContext,
		IsLocal:        false,
		CostTier:       tier,
	}
}

func matchClaude(lower string) (string, types.CostTier, bool) {
	switch {
	case strings.Contains(lower, "opus-4-5") || strings.Contains(lower, "opus-4.5"):
		return "Opus 4.5", types.CostTierOpus, true
	case strings.Contains(lower, "opus-4-6") || strings.Contains(lower, "opus-4.6"):
		return "Opus 4.6", types.CostTierOpus, true
	case strings.Contains(lower, "sonnet-4-5") || strings.Contains(lower, "sonnet-4.5"):
		return "Sonnet 4.5", types.CostTierSonnet, true
	// 3.5 patterns before generic 4/3 to avoid false matches
	case strings.Contains(lower, "3-5-sonnet") || strings.Contains(lower, "sonnet-3.5") || strings.Contains(lower, "sonnet-3-5"):
		return "Sonnet 3.5", types.CostTierSonnet, true
	case strings.Contains(lower, "sonnet-4"):
		return "Sonnet 4", types.CostTierSonnet, true
	case strings.Contains(lower, "3-opus") || strings.Contains(lower, "opus-3"):
		return "Opus 3", types.CostTierOpus, true
	case strings.Contains(lower, "3-5-haiku") || strings.Contains(lower, "haiku-3.5") || strings.Contains(lower, "haiku-3-5"):
		return "Haiku 3.5", types.CostTierHaiku, true
	case strings.Contains(lower, "3-haiku") || strings.Contains(lower, "haiku-3"):
		return "Haiku 3", types.CostTierHaiku, true
	case strings.Contains(lower, "haiku-4"):
		return "Haiku 4", types.CostTierHaiku, true
	}
	return "", types.CostTierSonnet, false
}

// detectCostTier determines cost tier from a display name string.
func detectCostTier(displayName string) types.CostTier {
	lower := strings.ToLower(displayName)
	switch {
	case strings.Contains(lower, "opus"):
		return types.CostTierOpus
	case strings.Contains(lower, "haiku"):
		return types.CostTierHaiku
	default:
		return types.CostTierSonnet
	}
}

func resolveOllama(modelID string, ollama ports.OllamaClient, cacheDir string) types.ModelInfo {
	// Strip prefix
	name := modelID
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.IndexByte(name, '/'); idx >= 0 {
		name = name[idx+1:]
	}

	short := shortenOllamaName(name)
	ctx := defaultLocalContext

	if ollama != nil {
		if size, err := ollama.GetContextSize(name); err == nil && size > 0 {
			ctx = size
		}
	}

	return types.ModelInfo{
		ShortName:      "🦙 " + short,
		DefaultContext: ctx,
		IsLocal:        true,
	}
}

func shortenOllamaName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "qwen3-coder"):
		return "Qwen3"
	case strings.HasPrefix(lower, "qwen2.5-coder"):
		return "Qwen2.5"
	case strings.HasPrefix(lower, "llama3"):
		return "Llama3"
	case strings.HasPrefix(lower, "llama2"):
		return "Llama2"
	case strings.HasPrefix(lower, "codellama"):
		return "CodeLlama"
	case strings.HasPrefix(lower, "mistral"):
		return "Mistral"
	case strings.HasPrefix(lower, "deepseek"):
		return "DeepSeek"
	case strings.HasPrefix(lower, "phi"):
		return "Phi"
	}
	// Truncate at first colon or dash after base name
	if idx := strings.Index(name, ":"); idx > 0 {
		return name[:idx]
	}
	return name
}

// OllamaHTTPClient implements ports.OllamaClient via HTTP.
type OllamaHTTPClient struct {
	BaseURL  string
	CacheDir string
	cache    ports.CacheStore
}

// NewOllamaClient creates an Ollama client.
func NewOllamaClient(cacheDir string, cache ports.CacheStore) *OllamaHTTPClient {
	return &OllamaHTTPClient{
		BaseURL:  "http://localhost:11434",
		CacheDir: cacheDir,
		cache:    cache,
	}
}

// GetContextSize queries Ollama for the model's context size.
func (c *OllamaHTTPClient) GetContextSize(model string) (int, error) {
	// Check cache first (30s TTL)
	cacheKey := strings.ReplaceAll(model, "/", "_")
	cacheKey = strings.ReplaceAll(cacheKey, ":", "_")
	cachePath := fmt.Sprintf("%s/ollama_ctx_%s.txt", c.CacheDir, cacheKey)

	if c.cache != nil {
		if data, fresh := c.cache.ReadIfFresh(cachePath, 30*time.Second); fresh {
			var size int
			if _, err := fmt.Sscanf(string(data), "%d", &size); err == nil && size > 0 {
				return size, nil
			}
		}
	}

	// Try /api/ps first
	if size, err := c.queryPS(model); err == nil && size > 0 {
		c.cacheSize(cachePath, size)
		return size, nil
	}

	// Try /api/show
	if size, err := c.queryShow(model); err == nil && size > 0 {
		c.cacheSize(cachePath, size)
		return size, nil
	}

	return 0, fmt.Errorf("could not determine context size for %s", model)
}

func (c *OllamaHTTPClient) queryPS(model string) (int, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(c.BaseURL + "/api/ps")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name          string `json:"name"`
			ContextLength int    `json:"context_length"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	for _, m := range result.Models {
		if strings.Contains(strings.ToLower(m.Name), strings.ToLower(model)) {
			return m.ContextLength, nil
		}
	}
	return 0, fmt.Errorf("model not found in /api/ps")
}

func (c *OllamaHTTPClient) queryShow(model string) (int, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	body := fmt.Sprintf(`{"name":"%s"}`, model)
	resp, err := client.Post(c.BaseURL+"/api/show", "application/json", strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		ModelInfo map[string]interface{} `json:"model_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	// Check known context_length fields
	for key, val := range result.ModelInfo {
		if strings.HasSuffix(key, ".context_length") {
			if f, ok := val.(float64); ok && f > 0 {
				return int(f), nil
			}
		}
	}

	return 0, fmt.Errorf("no context_length in model info")
}

func (c *OllamaHTTPClient) cacheSize(path string, size int) {
	if c.cache != nil {
		c.cache.WriteFile(path, []byte(fmt.Sprintf("%d", size)))
	}
}
