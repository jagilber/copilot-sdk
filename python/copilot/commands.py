# --------------------------------------------------------------------------------------------
#  Copyright (c) Microsoft Corporation. All rights reserved.
# --------------------------------------------------------------------------------------------

"""
Slash command support for the Copilot SDK.

Provides programmatic access to slash commands (e.g., /help, /model, /clear)
that were previously only available in the Copilot CLI's interactive TUI.
"""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class SlashCommandParameter:
    """Describes a parameter accepted by a slash command."""

    name: str
    """Parameter name"""
    description: str
    """Human-readable description"""
    required: bool
    """Whether this parameter is required"""


@dataclass
class SlashCommand:
    """Describes a slash command that can be executed via the SDK."""

    name: str
    """Command name including leading slash (e.g., '/help', '/model')"""
    description: str
    """Human-readable description of what the command does"""
    parameters: list[SlashCommandParameter] = field(default_factory=list)
    """Optional parameters the command accepts"""


@dataclass
class SlashCommandResult:
    """Result of executing a slash command."""

    handled: bool
    """Whether the command was handled"""
    method: str
    """How the command was executed: 'rpc', 'local', or 'passthrough'"""
    output: str | None = None
    """Optional output text from the command"""


@dataclass
class ParsedCommand:
    """Parsed slash command line."""

    name: str
    """Command name including leading slash"""
    args: list[str]
    """Arguments after the command name"""


BUILTIN_COMMANDS: list[SlashCommand] = [
    SlashCommand(
        name="/help",
        description="Show available slash commands and their descriptions",
    ),
    SlashCommand(
        name="/model",
        description="Switch the active model for this session",
        parameters=[
            SlashCommandParameter(
                name="modelId",
                description="Model identifier to switch to",
                required=True,
            ),
        ],
    ),
    SlashCommand(
        name="/compact",
        description="Compact the conversation history to free context window space",
    ),
    SlashCommand(
        name="/clear",
        description="Clear the current conversation context",
    ),
    SlashCommand(
        name="/agent",
        description="Select or deselect a custom agent",
        parameters=[
            SlashCommandParameter(
                name="action",
                description='"select" or "deselect"',
                required=True,
            ),
            SlashCommandParameter(
                name="name",
                description="Agent identifier (required for select)",
                required=False,
            ),
        ],
    ),
    SlashCommand(name="/review", description="Request a code review"),
    SlashCommand(name="/sessions", description="List or manage sessions"),
    SlashCommand(
        name="/plugin",
        description="Manage plugins (install, update, uninstall, marketplace)",
    ),
    SlashCommand(name="/skills", description="List available skills"),
    SlashCommand(name="/mcp", description="Manage MCP servers"),
    SlashCommand(name="/login", description="Log in to GitHub"),
    SlashCommand(name="/logout", description="Log out of GitHub"),
    SlashCommand(name="/context", description="Manage conversation context"),
    SlashCommand(name="/diff", description="Show differences in modified files"),
    SlashCommand(name="/tasks", description="List background tasks"),
]

_RPC_MAPPED_COMMANDS = frozenset(["/model", "/compact", "/agent"])


class CommandRegistry:
    """Registry of available slash commands with lookup and parsing utilities."""

    def __init__(self) -> None:
        self._commands: dict[str, SlashCommand] = {}
        for cmd in BUILTIN_COMMANDS:
            self._commands[cmd.name] = cmd

    def list(self) -> list[SlashCommand]:
        """Returns all registered slash commands."""
        return list(self._commands.values())

    def get(self, name: str) -> SlashCommand | None:
        """Returns a command by name, or None if not found."""
        return self._commands.get(name)

    def has(self, name: str) -> bool:
        """Returns True if a command with the given name exists."""
        return name in self._commands

    def has_rpc_mapping(self, name: str) -> bool:
        """Returns True if the command has a direct RPC method mapping."""
        return name in _RPC_MAPPED_COMMANDS

    def parse_command_line(self, input_str: str) -> ParsedCommand | None:
        """
        Parses a command line string into a command name and arguments.

        Args:
            input_str: Raw input string (e.g., "/model gpt-4")

        Returns:
            Parsed command with name and args, or None if not a slash command
        """
        trimmed = input_str.strip()
        if not trimmed.startswith("/"):
            return None

        parts = trimmed.split()
        return ParsedCommand(name=parts[0], args=parts[1:])
