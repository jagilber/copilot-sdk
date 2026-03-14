/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *--------------------------------------------------------------------------------------------*/

using GitHub.Copilot.SDK.Rpc;
using System.Text;
using System.Text.Json;

namespace GitHub.Copilot.SDK;

/// <summary>
/// Describes a parameter accepted by a slash command.
/// </summary>
public sealed record SlashCommandParameter(string Name, string Description, bool Required);

/// <summary>
/// Describes a slash command that can be executed via the SDK.
/// </summary>
public sealed record SlashCommand(string Name, string Description, SlashCommandParameter[]? Parameters = null);

/// <summary>
/// Result of executing a slash command.
/// </summary>
public sealed record SlashCommandResult
{
    /// <summary>Whether the command was handled.</summary>
    public bool Handled { get; init; }

    /// <summary>
    /// How the command was executed: "rpc", "local", or "passthrough".
    /// </summary>
    public string Method { get; init; } = string.Empty;

    /// <summary>Optional output text from the command.</summary>
    public string? Output { get; init; }
}

/// <summary>
/// Registry of available slash commands with lookup and parsing utilities.
/// </summary>
public sealed class CommandRegistry
{
    private static readonly SlashCommand[] _builtinCommands =
    [
        new("/help", "Show available slash commands and their descriptions"),
        new("/model", "Switch the active model for this session",
            [new("modelId", "Model identifier to switch to", true)]),
        new("/compact", "Compact the conversation history to free context window space"),
        new("/clear", "Clear the current conversation context"),
        new("/agent", "Select or deselect a custom agent",
        [
            new("action", "\"select\" or \"deselect\"", true),
            new("agentSlug", "Agent identifier (required for select)", false)
        ]),
        new("/review", "Request a code review"),
        new("/sessions", "List or manage sessions"),
        new("/plugin", "Manage plugins (install, update, uninstall, marketplace)"),
        new("/skills", "List available skills"),
        new("/mcp", "Manage MCP servers"),
        new("/login", "Log in to GitHub"),
        new("/logout", "Log out of GitHub"),
        new("/context", "Manage conversation context"),
        new("/diff", "Show differences in modified files"),
        new("/tasks", "List background tasks"),
    ];

    private static readonly HashSet<string> _rpcMappedCommands = ["/model", "/compact", "/agent"];

    private readonly Dictionary<string, SlashCommand> _commands = new();

    /// <summary>
    /// Creates a new CommandRegistry with all builtin commands.
    /// </summary>
    public CommandRegistry()
    {
        foreach (var cmd in _builtinCommands)
        {
            _commands[cmd.Name] = cmd;
        }
    }

    /// <summary>Returns all registered slash commands.</summary>
    public static IReadOnlyList<SlashCommand> List() => _builtinCommands;

    /// <summary>Returns a command by name, or null if not found.</summary>
    public SlashCommand? Get(string name) => _commands.GetValueOrDefault(name);

    /// <summary>Returns true if a command with the given name exists.</summary>
    public bool Has(string name) => _commands.ContainsKey(name);

    /// <summary>Returns true if the command has a direct RPC method mapping.</summary>
    public static bool HasRpcMapping(string name) => _rpcMappedCommands.Contains(name);

    /// <summary>
    /// Parses a command line string into a command name and arguments.
    /// </summary>
    public static (string Name, string[] Args)? ParseCommandLine(string input)
    {
        var trimmed = input.Trim();
        if (string.IsNullOrEmpty(trimmed) || !trimmed.StartsWith('/'))
        {
            return null;
        }

        var parts = trimmed.Split(' ', StringSplitOptions.RemoveEmptyEntries);
        return (parts[0], parts.Length > 1 ? parts[1..] : []);
    }
}

/// <summary>
/// Extension methods for CopilotSession slash command support.
/// </summary>
public static class SlashCommandExtensions
{
    private static readonly CommandRegistry _registry = new();

    /// <summary>
    /// Returns the list of supported slash commands.
    /// </summary>
    public static IReadOnlyList<SlashCommand> SupportedCommands(this CopilotSession session)
    {
        return CommandRegistry.List();
    }

    /// <summary>
    /// Executes a slash command.
    /// </summary>
    /// <param name="session">The session to execute the command on.</param>
    /// <param name="command">The slash command name or full command line.</param>
    /// <param name="args">Optional arguments for the command.</param>
    /// <param name="cancellationToken">Cancellation token.</param>
    /// <returns>A task that resolves with the command result.</returns>
    /// <exception cref="ArgumentException">If the command is empty or doesn't start with "/".</exception>
    public static async Task<SlashCommandResult> SendCommandAsync(
        this CopilotSession session,
        string command,
        string[]? args = null,
        CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrWhiteSpace(command))
        {
            throw new ArgumentException("Invalid command: command string cannot be empty", nameof(command));
        }

        var trimmed = command.Trim();
        if (!trimmed.StartsWith('/'))
        {
            throw new ArgumentException(
                "Invalid command: must start with \"/\" (e.g., \"/help\", \"/model gpt-4\")",
                nameof(command));
        }

        var parsed = CommandRegistry.ParseCommandLine(trimmed);
        var commandName = parsed!.Value.Name;
        var combinedArgs = parsed.Value.Args.Concat(args ?? []).ToArray();

        return commandName switch
        {
            "/help" => HandleHelpCommand(),
            "/clear" => HandleClearCommand(),
            "/model" => await HandleModelCommandAsync(session, combinedArgs, cancellationToken),
            "/compact" => await HandleCompactCommandAsync(session, cancellationToken),
            "/agent" => await HandleAgentCommandAsync(session, combinedArgs, cancellationToken),
            _ => await HandlePassthroughCommandAsync(session, commandName, combinedArgs, cancellationToken),
        };
    }

    private static SlashCommandResult HandleHelpCommand()
    {
        var commands = CommandRegistry.List();
        var sb = new StringBuilder();
        sb.AppendLine("Available slash commands:");
        sb.AppendLine();
        foreach (var cmd in commands)
        {
            sb.Append($"  {cmd.Name}");
            if (cmd.Parameters is { Length: > 0 })
            {
                foreach (var p in cmd.Parameters)
                {
                    sb.Append(p.Required ? $" <{p.Name}>" : $" [{p.Name}]");
                }
            }
            sb.AppendLine($" — {cmd.Description}");
        }
        return new() { Handled = true, Method = "local", Output = sb.ToString().TrimEnd() };
    }

    private static SlashCommandResult HandleClearCommand()
    {
        return new()
        {
            Handled = true,
            Method = "local",
            Output = "Session context cleared. Start a new session with CreateSessionAsync() for a fresh conversation."
        };
    }

    private static async Task<SlashCommandResult> HandleModelCommandAsync(
        CopilotSession session, string[] args, CancellationToken cancellationToken)
    {
        if (args.Length == 0)
        {
            var current = await session.Rpc.Model.GetCurrentAsync(cancellationToken);
            return new()
            {
                Handled = true,
                Method = "rpc",
                Output = $"Current model: {current.ModelId ?? "unknown"}"
            };
        }
        await session.Rpc.Model.SwitchToAsync(args[0], cancellationToken: cancellationToken);
        return new() { Handled = true, Method = "rpc", Output = $"Model switched to {args[0]}" };
    }

    private static async Task<SlashCommandResult> HandleCompactCommandAsync(
        CopilotSession session, CancellationToken cancellationToken)
    {
        await session.Rpc.Compaction.CompactAsync(cancellationToken);
        return new() { Handled = true, Method = "rpc", Output = "Conversation history compacted" };
    }

    private static async Task<SlashCommandResult> HandleAgentCommandAsync(
        CopilotSession session, string[] args, CancellationToken cancellationToken)
    {
        if (args.Length == 0 || args[0] == "list")
        {
            var agents = await session.Rpc.Agent.ListAsync(cancellationToken);
            return new()
            {
                Handled = true,
                Method = "rpc",
                Output = JsonSerializer.Serialize(agents, RpcJsonContext.Default.SessionAgentListResult)
            };
        }

        if (args[0] == "select" && args.Length >= 2)
        {
            await session.Rpc.Agent.SelectAsync(args[1], cancellationToken);
            return new() { Handled = true, Method = "rpc", Output = $"Agent '{args[1]}' selected" };
        }

        if (args[0] == "deselect")
        {
            await session.Rpc.Agent.DeselectAsync(cancellationToken);
            return new() { Handled = true, Method = "rpc", Output = "Agent deselected" };
        }

        return await HandlePassthroughCommandAsync(session, "/agent", args, cancellationToken);
    }

    private static async Task<SlashCommandResult> HandlePassthroughCommandAsync(
        CopilotSession session, string commandName, string[] args, CancellationToken cancellationToken)
    {
        var prompt = args.Length > 0
            ? $"{commandName} {string.Join(" ", args)}"
            : commandName;
        await session.SendAsync(new MessageOptions { Prompt = prompt }, cancellationToken);
        return new() { Handled = true, Method = "passthrough" };
    }
}
