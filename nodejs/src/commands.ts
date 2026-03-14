/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *--------------------------------------------------------------------------------------------*/

/**
 * Slash command support for the Copilot SDK.
 *
 * Provides programmatic access to slash commands (e.g., /help, /model, /clear)
 * that were previously only available in the Copilot CLI's interactive TUI.
 *
 * @module commands
 */

/**
 * Describes a parameter accepted by a slash command.
 */
export interface SlashCommandParameter {
    /** Parameter name */
    name: string;
    /** Human-readable description */
    description: string;
    /** Whether this parameter is required */
    required: boolean;
}

/**
 * Describes a slash command that can be executed via the SDK.
 */
export interface SlashCommand {
    /** Command name including leading slash (e.g., "/help", "/model") */
    name: string;
    /** Human-readable description of what the command does */
    description: string;
    /** Optional parameters the command accepts */
    parameters?: SlashCommandParameter[];
}

/**
 * Result of executing a slash command.
 */
export interface SlashCommandResult {
    /** Whether the command was handled */
    handled: boolean;
    /**
     * How the command was executed:
     * - "rpc": Mapped to an existing JSON-RPC method
     * - "local": Handled locally by the SDK (e.g., /help, /clear)
     * - "passthrough": Sent as a text prompt to the CLI
     */
    method: "rpc" | "local" | "passthrough";
    /** Optional output text from the command */
    output?: string;
}

/**
 * Parsed slash command line.
 */
export interface ParsedCommand {
    /** Command name including leading slash */
    name: string;
    /** Arguments after the command name */
    args: string[];
}

/**
 * Built-in slash commands supported by the Copilot CLI.
 *
 * Commands with RPC equivalents are handled natively via the SDK's
 * existing RPC methods. Others are sent as text prompts to the CLI.
 */
export const BUILTIN_COMMANDS: SlashCommand[] = [
    {
        name: "/help",
        description: "Show available slash commands and their descriptions",
    },
    {
        name: "/model",
        description: "Switch the active model for this session",
        parameters: [
            { name: "modelId", description: "Model identifier to switch to", required: true },
        ],
    },
    {
        name: "/compact",
        description: "Compact the conversation history to free context window space",
    },
    {
        name: "/clear",
        description: "Clear the current conversation context",
    },
    {
        name: "/agent",
        description: "Select or deselect a custom agent",
        parameters: [
            {
                name: "action",
                description: '"select" or "deselect"',
                required: true,
            },
            {
                name: "name",
                description: "Agent identifier (required for select)",
                required: false,
            },
        ],
    },
    {
        name: "/review",
        description: "Request a code review",
    },
    {
        name: "/sessions",
        description: "List or manage sessions",
    },
    {
        name: "/plugin",
        description: "Manage plugins (install, update, uninstall, marketplace)",
    },
    {
        name: "/skills",
        description: "List available skills",
    },
    {
        name: "/mcp",
        description: "Manage MCP servers",
    },
    {
        name: "/login",
        description: "Log in to GitHub",
    },
    {
        name: "/logout",
        description: "Log out of GitHub",
    },
    {
        name: "/context",
        description: "Manage conversation context",
    },
    {
        name: "/diff",
        description: "Show differences in modified files",
    },
    {
        name: "/tasks",
        description: "List background tasks",
    },
];

/** Set of command names that have direct RPC method equivalents */
const RPC_MAPPED_COMMANDS = new Set(["/model", "/compact", "/agent"]);

/**
 * Registry of available slash commands with lookup and parsing utilities.
 *
 * @example
 * ```typescript
 * const registry = new CommandRegistry();
 * const commands = registry.list();
 * const modelCmd = registry.get("/model");
 * const parsed = registry.parseCommandLine("/model gpt-4");
 * ```
 */
export class CommandRegistry {
    private commands: Map<string, SlashCommand>;

    constructor() {
        this.commands = new Map();
        for (const cmd of BUILTIN_COMMANDS) {
            this.commands.set(cmd.name, cmd);
        }
    }

    /** Returns all registered slash commands. */
    list(): SlashCommand[] {
        return [...this.commands.values()];
    }

    /** Returns a command by name, or undefined if not found. */
    get(name: string): SlashCommand | undefined {
        return this.commands.get(name);
    }

    /** Returns true if a command with the given name exists. */
    has(name: string): boolean {
        return this.commands.has(name);
    }

    /** Returns true if the command has a direct RPC method mapping. */
    hasRpcMapping(name: string): boolean {
        return RPC_MAPPED_COMMANDS.has(name);
    }

    /**
     * Parses a command line string into a command name and arguments.
     *
     * @param input - Raw input string (e.g., "/model gpt-4")
     * @returns Parsed command with name and args, or null if not a slash command
     */
    parseCommandLine(input: string): ParsedCommand | null {
        const trimmed = input.trim();
        if (!trimmed.startsWith("/")) {
            return null;
        }

        const parts = trimmed.split(/\s+/);
        return {
            name: parts[0],
            args: parts.slice(1),
        };
    }
}
