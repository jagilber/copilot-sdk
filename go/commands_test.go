/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *--------------------------------------------------------------------------------------------*/

package copilot

import (
	"testing"
)

func TestBuiltinCommands(t *testing.T) {
	t.Run("contains expected commands", func(t *testing.T) {
		names := make(map[string]bool)
		for _, cmd := range builtinCommands {
			names[cmd.Name] = true
		}

		for _, expected := range []string{"/help", "/model", "/compact", "/clear", "/agent"} {
			if !names[expected] {
				t.Errorf("expected builtin command %q not found", expected)
			}
		}
	})

	t.Run("all commands start with slash", func(t *testing.T) {
		for _, cmd := range builtinCommands {
			if cmd.Name[0] != '/' {
				t.Errorf("command %q does not start with /", cmd.Name)
			}
			if cmd.Description == "" {
				t.Errorf("command %q has empty description", cmd.Name)
			}
		}
	})
}

func TestCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	t.Run("List returns all builtin commands", func(t *testing.T) {
		commands := registry.List()
		if len(commands) != len(builtinCommands) {
			t.Errorf("expected %d commands, got %d", len(builtinCommands), len(commands))
		}
	})

	t.Run("Get returns command by name", func(t *testing.T) {
		cmd := registry.Get("/help")
		if cmd == nil {
			t.Fatal("expected /help command, got nil")
		}
		if cmd.Name != "/help" {
			t.Errorf("expected name /help, got %s", cmd.Name)
		}
	})

	t.Run("Get returns nil for unknown command", func(t *testing.T) {
		cmd := registry.Get("/nonexistent")
		if cmd != nil {
			t.Errorf("expected nil for unknown command, got %v", cmd)
		}
	})

	t.Run("Has returns true for known commands", func(t *testing.T) {
		if !registry.Has("/help") {
			t.Error("expected Has(/help) to be true")
		}
		if !registry.Has("/model") {
			t.Error("expected Has(/model) to be true")
		}
	})

	t.Run("Has returns false for unknown commands", func(t *testing.T) {
		if registry.Has("/nonexistent") {
			t.Error("expected Has(/nonexistent) to be false")
		}
	})

	t.Run("HasRpcMapping for RPC commands", func(t *testing.T) {
		if !registry.HasRpcMapping("/model") {
			t.Error("expected HasRpcMapping(/model) to be true")
		}
		if !registry.HasRpcMapping("/compact") {
			t.Error("expected HasRpcMapping(/compact) to be true")
		}
		if !registry.HasRpcMapping("/agent") {
			t.Error("expected HasRpcMapping(/agent) to be true")
		}
	})

	t.Run("HasRpcMapping false for passthrough commands", func(t *testing.T) {
		if registry.HasRpcMapping("/help") {
			t.Error("expected HasRpcMapping(/help) to be false")
		}
		if registry.HasRpcMapping("/nonexistent") {
			t.Error("expected HasRpcMapping(/nonexistent) to be false")
		}
	})

	t.Run("ParseCommandLine basic", func(t *testing.T) {
		result := registry.ParseCommandLine("/help")
		if result == nil {
			t.Fatal("expected parsed result, got nil")
		}
		if result.Name != "/help" {
			t.Errorf("expected name /help, got %s", result.Name)
		}
		if len(result.Args) != 0 {
			t.Errorf("expected 0 args, got %d", len(result.Args))
		}
	})

	t.Run("ParseCommandLine with args", func(t *testing.T) {
		result := registry.ParseCommandLine("/model gpt-4")
		if result == nil {
			t.Fatal("expected parsed result, got nil")
		}
		if result.Name != "/model" {
			t.Errorf("expected name /model, got %s", result.Name)
		}
		if len(result.Args) != 1 || result.Args[0] != "gpt-4" {
			t.Errorf("expected args [gpt-4], got %v", result.Args)
		}
	})

	t.Run("ParseCommandLine multiple args", func(t *testing.T) {
		result := registry.ParseCommandLine("/agent select my-agent")
		if result == nil {
			t.Fatal("expected parsed result, got nil")
		}
		if result.Name != "/agent" {
			t.Errorf("expected name /agent, got %s", result.Name)
		}
		if len(result.Args) != 2 || result.Args[0] != "select" || result.Args[1] != "my-agent" {
			t.Errorf("expected args [select my-agent], got %v", result.Args)
		}
	})

	t.Run("ParseCommandLine with whitespace", func(t *testing.T) {
		result := registry.ParseCommandLine("  /help  ")
		if result == nil {
			t.Fatal("expected parsed result, got nil")
		}
		if result.Name != "/help" {
			t.Errorf("expected name /help, got %s", result.Name)
		}
	})

	t.Run("ParseCommandLine non-command returns nil", func(t *testing.T) {
		result := registry.ParseCommandLine("hello world")
		if result != nil {
			t.Errorf("expected nil for non-command, got %v", result)
		}

		result = registry.ParseCommandLine("")
		if result != nil {
			t.Errorf("expected nil for empty string, got %v", result)
		}
	})
}

func TestSession_SupportedCommands(t *testing.T) {
	session, cleanup := newTestSession()
	defer cleanup()

	commands := session.SupportedCommands()
	if len(commands) == 0 {
		t.Fatal("expected non-empty commands list")
	}

	for _, cmd := range commands {
		if cmd.Name[0] != '/' {
			t.Errorf("command %q does not start with /", cmd.Name)
		}
		if cmd.Description == "" {
			t.Errorf("command %q has empty description", cmd.Name)
		}
	}
}

func TestSession_SendCommand_Validation(t *testing.T) {
	session, cleanup := newTestSession()
	defer cleanup()

	t.Run("empty command returns error", func(t *testing.T) {
		_, err := session.SendCommand(t.Context(), "")
		if err == nil {
			t.Error("expected error for empty command")
		}
	})

	t.Run("non-slash command returns error", func(t *testing.T) {
		_, err := session.SendCommand(t.Context(), "hello")
		if err == nil {
			t.Error("expected error for non-slash command")
		}
	})

	t.Run("help command returns local result", func(t *testing.T) {
		result, err := session.SendCommand(t.Context(), "/help")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Handled {
			t.Error("expected handled to be true")
		}
		if result.Method != "local" {
			t.Errorf("expected method local, got %s", result.Method)
		}
		if result.Output == "" {
			t.Error("expected non-empty output for /help")
		}
	})

	t.Run("clear command returns local result", func(t *testing.T) {
		result, err := session.SendCommand(t.Context(), "/clear")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Handled {
			t.Error("expected handled to be true")
		}
		if result.Method != "local" {
			t.Errorf("expected method local, got %s", result.Method)
		}
	})
}
