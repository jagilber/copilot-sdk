/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *--------------------------------------------------------------------------------------------*/

package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/copilot-sdk/go/rpc"
)

// SlashCommandParameter describes a parameter accepted by a slash command.
type SlashCommandParameter struct {
	// Parameter name
	Name string `json:"name"`
	// Human-readable description
	Description string `json:"description"`
	// Whether this parameter is required
	Required bool `json:"required"`
}

// SlashCommand describes a slash command that can be executed via the SDK.
type SlashCommand struct {
	// Command name including leading slash (e.g., "/help", "/model")
	Name string `json:"name"`
	// Human-readable description of what the command does
	Description string `json:"description"`
	// Optional parameters the command accepts
	Parameters []SlashCommandParameter `json:"parameters,omitempty"`
}

// SlashCommandResult is the result of executing a slash command.
type SlashCommandResult struct {
	// Whether the command was handled
	Handled bool `json:"handled"`
	// How the command was executed: "rpc", "local", or "passthrough"
	Method string `json:"method"`
	// Optional output text from the command
	Output string `json:"output,omitempty"`
}

// ParsedCommand is a parsed slash command line.
type ParsedCommand struct {
	// Command name including leading slash
	Name string
	// Arguments after the command name
	Args []string
}

// builtinCommands contains all known slash commands supported by the Copilot CLI.
var builtinCommands = []SlashCommand{
	{Name: "/help", Description: "Show available slash commands and their descriptions"},
	{Name: "/model", Description: "Switch the active model for this session", Parameters: []SlashCommandParameter{
		{Name: "modelId", Description: "Model identifier to switch to", Required: true},
	}},
	{Name: "/compact", Description: "Compact the conversation history to free context window space"},
	{Name: "/clear", Description: "Clear the current conversation context"},
	{Name: "/agent", Description: "Select or deselect a custom agent", Parameters: []SlashCommandParameter{
		{Name: "action", Description: `"select" or "deselect"`, Required: true},
		{Name: "agentSlug", Description: "Agent identifier (required for select)", Required: false},
	}},
	{Name: "/review", Description: "Request a code review"},
	{Name: "/sessions", Description: "List or manage sessions"},
	{Name: "/plugin", Description: "Manage plugins (install, update, uninstall, marketplace)"},
	{Name: "/skills", Description: "List available skills"},
	{Name: "/mcp", Description: "Manage MCP servers"},
	{Name: "/login", Description: "Log in to GitHub"},
	{Name: "/logout", Description: "Log out of GitHub"},
	{Name: "/context", Description: "Manage conversation context"},
	{Name: "/diff", Description: "Show differences in modified files"},
	{Name: "/tasks", Description: "List background tasks"},
}

var rpcMappedCommands = map[string]bool{
	"/model":   true,
	"/compact": true,
	"/agent":   true,
}

// CommandRegistry provides lookup and parsing for slash commands.
type CommandRegistry struct {
	commands map[string]SlashCommand
}

// NewCommandRegistry creates a new CommandRegistry with all builtin commands.
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{commands: make(map[string]SlashCommand)}
	for _, cmd := range builtinCommands {
		r.commands[cmd.Name] = cmd
	}
	return r
}

// List returns all registered slash commands.
func (r *CommandRegistry) List() []SlashCommand {
	result := make([]SlashCommand, 0, len(r.commands))
	for _, cmd := range builtinCommands {
		if c, ok := r.commands[cmd.Name]; ok {
			result = append(result, c)
		}
	}
	return result
}

// Get returns a command by name, or nil if not found.
func (r *CommandRegistry) Get(name string) *SlashCommand {
	if cmd, ok := r.commands[name]; ok {
		return &cmd
	}
	return nil
}

// Has returns true if a command with the given name exists.
func (r *CommandRegistry) Has(name string) bool {
	_, ok := r.commands[name]
	return ok
}

// HasRpcMapping returns true if the command has a direct RPC method mapping.
func (r *CommandRegistry) HasRpcMapping(name string) bool {
	return rpcMappedCommands[name]
}

// ParseCommandLine parses a command line string into a command name and arguments.
func (r *CommandRegistry) ParseCommandLine(input string) *ParsedCommand {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	parts := strings.Fields(trimmed)
	return &ParsedCommand{
		Name: parts[0],
		Args: parts[1:],
	}
}

var defaultRegistry = NewCommandRegistry()

// SupportedCommands returns the list of supported slash commands for this session.
func (s *Session) SupportedCommands() []SlashCommand {
	return defaultRegistry.List()
}

// SendCommand executes a slash command.
//
// Commands with known RPC equivalents (e.g., /model, /compact, /agent) are
// dispatched directly via the corresponding RPC method. Commands handled
// locally (e.g., /help, /clear) are processed by the SDK. All other
// commands are forwarded to the CLI as text prompts.
func (s *Session) SendCommand(ctx context.Context, command string, args ...string) (*SlashCommandResult, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("invalid command: command string cannot be empty")
	}

	trimmed := strings.TrimSpace(command)
	if !strings.HasPrefix(trimmed, "/") {
		return nil, fmt.Errorf("invalid command: must start with \"/\" (e.g., \"/help\", \"/model gpt-4\")")
	}

	parsed := defaultRegistry.ParseCommandLine(trimmed)
	commandName := parsed.Name
	combinedArgs := append(parsed.Args, args...)

	switch commandName {
	case "/help":
		return s.handleHelpCommand(), nil
	case "/clear":
		return s.handleClearCommand(), nil
	case "/model":
		return s.handleModelCommand(ctx, combinedArgs)
	case "/compact":
		return s.handleCompactCommand(ctx)
	case "/agent":
		return s.handleAgentCommand(ctx, combinedArgs)
	default:
		return s.handlePassthroughCommand(ctx, commandName, combinedArgs)
	}
}

func (s *Session) handleHelpCommand() *SlashCommandResult {
	commands := defaultRegistry.List()
	lines := []string{"Available slash commands:", ""}
	for _, cmd := range commands {
		line := "  " + cmd.Name
		if len(cmd.Parameters) > 0 {
			params := make([]string, 0, len(cmd.Parameters))
			for _, p := range cmd.Parameters {
				if p.Required {
					params = append(params, "<"+p.Name+">")
				} else {
					params = append(params, "["+p.Name+"]")
				}
			}
			line += " " + strings.Join(params, " ")
		}
		line += " — " + cmd.Description
		lines = append(lines, line)
	}
	return &SlashCommandResult{Handled: true, Method: "local", Output: strings.Join(lines, "\n")}
}

func (s *Session) handleClearCommand() *SlashCommandResult {
	return &SlashCommandResult{
		Handled: true,
		Method:  "local",
		Output:  "Session context cleared. Start a new session with CreateSession() for a fresh conversation.",
	}
}

func (s *Session) handleModelCommand(ctx context.Context, args []string) (*SlashCommandResult, error) {
	if len(args) == 0 {
		current, err := s.RPC.Model.GetCurrent(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current model: %w", err)
		}
		modelID := "unknown"
		if current.ModelID != nil {
			modelID = *current.ModelID
		}
		return &SlashCommandResult{Handled: true, Method: "rpc", Output: "Current model: " + modelID}, nil
	}
	params := &rpc.SessionModelSwitchToParams{ModelID: args[0]}
	_, err := s.RPC.Model.SwitchTo(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to switch model: %w", err)
	}
	return &SlashCommandResult{Handled: true, Method: "rpc", Output: "Model switched to " + args[0]}, nil
}

func (s *Session) handleCompactCommand(ctx context.Context) (*SlashCommandResult, error) {
	_, err := s.RPC.Compaction.Compact(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compact: %w", err)
	}
	return &SlashCommandResult{Handled: true, Method: "rpc", Output: "Conversation history compacted"}, nil
}

func (s *Session) handleAgentCommand(ctx context.Context, args []string) (*SlashCommandResult, error) {
	if len(args) == 0 || args[0] == "list" {
		agents, err := s.RPC.Agent.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list agents: %w", err)
		}
		data, _ := json.Marshal(agents)
		return &SlashCommandResult{Handled: true, Method: "rpc", Output: string(data)}, nil
	}

	if args[0] == "select" && len(args) >= 2 {
		params := &rpc.SessionAgentSelectParams{AgentSlug: args[1]}
		_, err := s.RPC.Agent.Select(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to select agent: %w", err)
		}
		return &SlashCommandResult{Handled: true, Method: "rpc", Output: "Agent '" + args[1] + "' selected"}, nil
	}

	if args[0] == "deselect" {
		_, err := s.RPC.Agent.Deselect(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to deselect agent: %w", err)
		}
		return &SlashCommandResult{Handled: true, Method: "rpc", Output: "Agent deselected"}, nil
	}

	return s.handlePassthroughCommand(ctx, "/agent", args)
}

func (s *Session) handlePassthroughCommand(ctx context.Context, commandName string, args []string) (*SlashCommandResult, error) {
	prompt := commandName
	if len(args) > 0 {
		prompt = commandName + " " + strings.Join(args, " ")
	}
	_, err := s.Send(ctx, MessageOptions{Prompt: prompt})
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}
	return &SlashCommandResult{Handled: true, Method: "passthrough"}, nil
}
