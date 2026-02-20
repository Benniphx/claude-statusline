package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	adaptapi "github.com/Benniphx/claude-statusline/adapter/api"
	"github.com/Benniphx/claude-statusline/adapter/cache"
	adaptconfig "github.com/Benniphx/claude-statusline/adapter/config"
	"github.com/Benniphx/claude-statusline/adapter/platform"
	adaptrender "github.com/Benniphx/claude-statusline/adapter/render"
	"github.com/Benniphx/claude-statusline/core/agents"
	corecontext "github.com/Benniphx/claude-statusline/core/context"
	"github.com/Benniphx/claude-statusline/core/cost"
	"github.com/Benniphx/claude-statusline/core/daemon"
	"github.com/Benniphx/claude-statusline/core/model"
	"github.com/Benniphx/claude-statusline/core/ollama"
	"github.com/Benniphx/claude-statusline/core/ratelimit"
	"github.com/Benniphx/claude-statusline/core/types"
	"github.com/Benniphx/claude-statusline/core/update"
)

var version = "dev"

func main() {
	// Handle flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version)
			return
		case "--daemon":
			runDaemon()
			return
		case "setup":
			runSetup()
			return
		}
	}

	// Wire adapters
	plat := platform.Detect()
	store := cache.New()
	api := adaptapi.New()
	rend := adaptrender.New()
	cfg := adaptconfig.Load()
	cfg.Version = version

	// Separator: 2 spaces + dim │ + 2 spaces (matching bash)
	sep := "  " + rend.Dim("│") + "  "

	// Parse stdin
	input, err := parseStdin()
	if err != nil || isEmptyInput(input) {
		// Empty input: dim "Starting..." + optional cached 5h rate
		fmt.Print(rend.Dim("Starting..."))
		creds, credErr := plat.GetCredentials()
		if credErr == nil && creds.HasOAuth() {
			rateStr := ratelimit.Render(creds, cfg, plat, store, api, rend)
			if rateStr != "" {
				fmt.Print(sep + rateStr)
			}
		}
		return
	}

	// Resolve model info
	ollamaClient := model.NewOllamaClient(cfg.CacheDir, store)
	modelInfo := model.Resolve(input.Model.ModelID, input.Model.DisplayName, ollamaClient, cfg.CacheDir)

	// Calculate context display
	ctxDisplay := corecontext.Calculate(input, modelInfo, cfg)

	// Get context-based color (used for model name)
	ctxColor := adaptrender.ColorForPercent(ctxDisplay.PercentUsed)

	// Get credentials
	creds, _ := plat.GetCredentials()

	// Build sections
	var sections []string

	// 1. Model name (colored by context %) + agent count
	modelSection := rend.Color(modelInfo.ShortName, ctxColor)
	agentInfo := agents.Count()
	if agentInfo.HasSubagents {
		modelSection += rend.Dim(fmt.Sprintf(" (%d)", agentInfo.Total))
	}
	sections = append(sections, modelSection)

	// 2. Context window
	ctxSection := corecontext.Render(ctxDisplay, cfg, rend)
	sections = append(sections, ctxSection)

	// 3. Rate limit sections (OAuth) or Cost sections (API key)
	if creds.HasOAuth() {
		rate := ratelimit.RenderSections(input, creds, cfg, plat, store, api, rend, modelInfo.CostTier)
		sections = append(sections, rate.FiveHour, rate.Burn, rate.SevenDay)
	} else {
		cs := cost.RenderSections(input, cfg, plat, store, rend, modelInfo.CostTier)
		sections = append(sections, cs.Session, cs.Daily, cs.Burn)
	}

	// 4. Duration
	durationSection := rend.Dim(fmt.Sprintf("%dm", ctxDisplay.DurationMin))

	// 5. Lines info (appended to duration with 1-space separator)
	linesInfo := ""
	if ctxDisplay.LinesAdded > 0 || ctxDisplay.LinesRemoved > 0 {
		linesInfo = fmt.Sprintf(" %s %s+%d%s/%s-%d%s",
			rend.Dim("│"),
			adaptrender.Green, ctxDisplay.LinesAdded, adaptrender.Reset,
			adaptrender.Red, ctxDisplay.LinesRemoved, adaptrender.Reset)
	}

	// 6. Ollama stats (only if stats file exists and is fresh)
	ollamaSection := ""
	if stats, err := ollama.ReadStats(ollama.StatsPath); err == nil {
		if rendered := ollama.Render(stats); rendered != "" {
			ollamaSection = sep + rend.Dim(rendered)
		}
	}

	// 7. Update notice
	updateNotice := update.Render(cfg.Version, cfg.CacheDir, store, api, rend)

	// Assemble: sections joined by sep, then duration+lines+ollama+update
	sections = append(sections, durationSection)
	output := strings.Join(sections, sep)
	output += linesInfo + ollamaSection + updateNotice
	fmt.Print(output)
}

func parseStdin() (types.Input, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return types.Input{}, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return types.Input{}, fmt.Errorf("empty input")
	}

	var input types.Input
	if err := json.Unmarshal(data, &input); err != nil {
		return types.Input{}, err
	}

	return input, nil
}

func isEmptyInput(input types.Input) bool {
	return input.Model.ModelID == "" && input.Model.DisplayName == "" &&
		input.ContextWindow.ContextWindowSize == 0 &&
		input.Cost.TotalDurationMS == 0
}

func runDaemon() {
	plat := platform.Detect()
	store := cache.New()
	api := adaptapi.New()
	cfg := adaptconfig.Load()

	creds, err := plat.GetCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: no credentials: %v\n", err)
		os.Exit(1)
	}

	if err := daemon.Run(cfg, creds, plat, store, api); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		os.Exit(1)
	}
}
