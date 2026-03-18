package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Benniphx/claude-statusline/adapter/cache"
	"github.com/Benniphx/claude-statusline/core/daemon"
	"github.com/Benniphx/claude-statusline/core/settings"
)

func runSetup() {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	binary := fs.String("binary", "", "path to the statusline binary")
	projectDir := fs.String("project-dir", ".", "project directory")
	pluginID := fs.String("plugin-id", "statusline@claude-statusline", "plugin identifier")

	if err := fs.Parse(os.Args[2:]); err != nil {
		printJSON("systemMessage", fmt.Sprintf("Statusline setup: %v", err))
		os.Exit(0)
	}

	if *binary == "" {
		printJSON("systemMessage", "Statusline setup: --binary is required.")
		os.Exit(0)
	}

	settingsPath, _ := settings.FindSettingsFile(*projectDir, *pluginID)
	store := cache.New()

	result, err := settings.Setup(settingsPath, *binary, store)
	if err != nil {
		printJSON("systemMessage", fmt.Sprintf("Statusline setup failed: %v", err))
		os.Exit(0)
	}

	// Auto-start background daemon or restart stale daemon (after plugin update).
	if daemon.IsRunning() {
		if daemon.IsStale(*binary) {
			daemon.Stop()
			startDaemon(*binary)
		}
	} else {
		startDaemon(*binary)
	}

	switch result.Action {
	case "noop":
		printJSON("statusMessage", "Statusline ready.")
	case "created":
		printJSON("statusMessage", "Statusline configured.")
	case "updated":
		printJSON("statusMessage", result.Message)
	}
}

// startDaemon launches the statusline daemon as a detached background process.
func startDaemon(binary string) {
	cmd := exec.Command(binary, "--daemon")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
}

func printJSON(key, value string) {
	data, _ := json.Marshal(map[string]string{key: value})
	fmt.Println(string(data))
}
