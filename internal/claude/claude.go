package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// IsClaudeToGoHook identifies if a command is a ClaudeToGo hook
func IsClaudeToGoHook(command string) bool {
	return strings.Contains(command, "claudetogo") && strings.Contains(command, "--hook")
}

// CleanupAllClaudeToGoHooks removes ClaudeToGo hooks from all hook types
// If a hook type contains only ClaudeToGo hooks, the entire type is removed
// If it contains mixed hooks, only ClaudeToGo hooks are filtered out
func CleanupAllClaudeToGoHooks(hooks map[string][]types.HookMatcher) {
	var keysToRemove []string

	for hookName, matchers := range hooks {
		var updatedMatchers []types.HookMatcher
		hasNonClaudeToGoHooks := false

		for _, matcher := range matchers {
			var preservedHooks []types.HookConfig

			for _, hook := range matcher.Hooks {
				if !IsClaudeToGoHook(hook.Command) {
					preservedHooks = append(preservedHooks, hook)
					hasNonClaudeToGoHooks = true
				}
			}

			if len(preservedHooks) > 0 {
				updatedMatchers = append(updatedMatchers, types.HookMatcher{
					Matcher: matcher.Matcher,
					Hooks:   preservedHooks,
				})
			}
		}

		if hasNonClaudeToGoHooks {
			// Keep the hook type but with ClaudeToGo hooks filtered out
			hooks[hookName] = updatedMatchers
		} else {
			// Mark for removal if it contained only ClaudeToGo hooks
			keysToRemove = append(keysToRemove, hookName)
		}
	}

	// Remove hook types that contained only ClaudeToGo hooks
	for _, key := range keysToRemove {
		delete(hooks, key)
	}
}

// BuildClaudeToGoCommand constructs the ClaudeToGo hook command string
func BuildClaudeToGoCommand(config types.ConfigFile) string {
	var cmd strings.Builder

	execPath, err := os.Executable()
	if err != nil {
		execPath = "./claudetogo"
	}

	cmd.WriteString(execPath)
	cmd.WriteString(" --hook")

	if config.LogFile != "claude-events.jsonl" {
		cmd.WriteString(fmt.Sprintf(` --logfile "%s"`, config.LogFile))
	}

	if config.Verbose {
		cmd.WriteString(" --verbose")
	}

	return cmd.String()
}

// UpdateHookType adds our ClaudeToGo hook to existing matchers (ClaudeToGo hooks already cleaned up)
func UpdateHookType(existingMatchers []types.HookMatcher, newCommand string, timeout int) []types.HookMatcher {
	var updatedMatchers []types.HookMatcher
	hasWildcardMatcher := false

	// Preserve all existing matchers and add our hook to wildcard matcher if it exists
	for _, matcher := range existingMatchers {
		if matcher.Matcher == "*" {
			hasWildcardMatcher = true
			// Add our hook to existing wildcard matcher
			updatedHooks := append(matcher.Hooks, types.HookConfig{
				Type:    "command",
				Command: newCommand,
				Timeout: &timeout,
			})
			updatedMatchers = append(updatedMatchers, types.HookMatcher{
				Matcher: matcher.Matcher,
				Hooks:   updatedHooks,
			})
		} else {
			// Preserve non-wildcard matchers as-is
			updatedMatchers = append(updatedMatchers, matcher)
		}
	}

	// If no wildcard matcher exists, create one with our hook
	if !hasWildcardMatcher {
		updatedMatchers = append(updatedMatchers, types.HookMatcher{
			Matcher: "*",
			Hooks: []types.HookConfig{
				{
					Type:    "command",
					Command: newCommand,
					Timeout: &timeout,
				},
			},
		})
	}

	return updatedMatchers
}

// LoadExistingSettings safely loads existing settings.json while preserving unknown fields
func LoadExistingSettings(path string) (*types.ClaudeSettingsConfig, error) {
	var settingsConfig types.ClaudeSettingsConfig

	if _, err := os.Stat(path); err != nil {
		// File doesn't exist, return empty config
		settingsConfig.Extra = make(map[string]json.RawMessage)
		return &settingsConfig, nil
	}

	// File exists, load it with full preservation
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open existing settings.json: %w", err)
	}
	defer file.Close()

	// First, read into a generic map to capture all fields
	var rawConfig map[string]json.RawMessage
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&rawConfig); err != nil {
		return nil, fmt.Errorf("could not parse existing settings.json: %w", err)
	}

	// Initialize Extra map
	settingsConfig.Extra = make(map[string]json.RawMessage)

	// Extract known fields and preserve unknown ones
	for key, value := range rawConfig {
		switch key {
		case "hooks":
			if err := json.Unmarshal(value, &settingsConfig.Hooks); err != nil {
				return nil, fmt.Errorf("could not parse hooks field: %w", err)
			}
		default:
			// Preserve unknown fields
			settingsConfig.Extra[key] = value
		}
	}

	return &settingsConfig, nil
}

// copyFile creates a backup copy of a file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// SaveSettingsWithPreservation safely saves settings while preserving unknown fields
func SaveSettingsWithPreservation(settingsConfig *types.ClaudeSettingsConfig, path string) error {
	// Create a map to hold the final JSON structure
	finalConfig := make(map[string]any)

	// Add preserved unknown fields first
	for key, value := range settingsConfig.Extra {
		var unmarshaled any
		if err := json.Unmarshal(value, &unmarshaled); err != nil {
			return fmt.Errorf("could not unmarshal preserved field %s: %w", key, err)
		}
		finalConfig[key] = unmarshaled
	}

	// Add hooks configuration (this will override any existing hooks)
	if len(settingsConfig.Hooks) > 0 {
		finalConfig["hooks"] = settingsConfig.Hooks
	}

	// Create backup of existing file
	if _, err := os.Stat(path); err == nil {
		backupPath := path + ".backup"
		if err := copyFile(path, backupPath); err != nil {
			// Log warning but don't fail
			log.Printf("[WARNING] Could not create backup at %s: %v", backupPath, err)
		}
	}

	// Write the merged configuration
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create settings.json: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(finalConfig); err != nil {
		return fmt.Errorf("could not write settings.json: %w", err)
	}

	return nil
}

// ConfigureHooksAtLocation configures Claude Code hooks at specified location
func ConfigureHooksAtLocation(config types.ConfigFile, location *types.ConfigLocation) error {
	// Ensure directory exists
	claudeDir := filepath.Dir(location.Path)
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", claudeDir, err)
	}

	// Build the command from config
	newCommand := BuildClaudeToGoCommand(config)
	timeout := 30

	// Load existing settings.json safely while preserving unknown fields
	settingsConfig, err := LoadExistingSettings(location.Path)
	if err != nil {
		return fmt.Errorf("could not load existing settings: %w", err)
	}

	// Initialize hooks if nil
	if settingsConfig.Hooks == nil {
		settingsConfig.Hooks = make(map[string][]types.HookMatcher)
	}

	// Clean up all ClaudeToGo hooks from all hook types before adding new ones
	CleanupAllClaudeToGoHooks(settingsConfig.Hooks)

	// Add our new ClaudeToGo hooks to target hook types
	targetHooks := []string{"Stop", "Notification"}
	for _, hookType := range targetHooks {
		settingsConfig.Hooks[hookType] = UpdateHookType(settingsConfig.Hooks[hookType], newCommand, timeout)
	}

	// Save the updated settings.json while preserving existing configuration
	if err := SaveSettingsWithPreservation(settingsConfig, location.Path); err != nil {
		return fmt.Errorf("could not save settings.json: %w", err)
	}

	fmt.Printf("✅ Claude Code hooks configured at: %s\n", location.Path)
	fmt.Printf("📋 Configuration scope: %s\n", location.Scope)

	return nil
}
