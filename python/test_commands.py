"""Tests for slash command support in the Copilot SDK."""

import pytest

from copilot.commands import BUILTIN_COMMANDS, CommandRegistry, SlashCommandResult


class TestSlashCommandTypes:
    """Tests for slash command type definitions."""

    def test_builtin_commands_contains_expected_commands(self) -> None:
        command_names = [c.name for c in BUILTIN_COMMANDS]

        assert "/help" in command_names
        assert "/model" in command_names
        assert "/compact" in command_names
        assert "/clear" in command_names
        assert "/agent" in command_names

        for cmd in BUILTIN_COMMANDS:
            assert cmd.name.startswith("/")
            assert cmd.description
            assert isinstance(cmd.description, str)

    def test_each_command_has_required_shape(self) -> None:
        for cmd in BUILTIN_COMMANDS:
            assert hasattr(cmd, "name")
            assert hasattr(cmd, "description")
            if cmd.parameters:
                assert isinstance(cmd.parameters, list)
                for param in cmd.parameters:
                    assert hasattr(param, "name")
                    assert hasattr(param, "description")
                    assert hasattr(param, "required")


class TestCommandRegistry:
    """Tests for the command registry."""

    def setup_method(self) -> None:
        self.registry = CommandRegistry()

    def test_list_returns_all_builtin_commands(self) -> None:
        commands = self.registry.list()
        assert len(commands) > 0
        assert len(commands) == len(BUILTIN_COMMANDS)

    def test_get_returns_command_by_name(self) -> None:
        cmd = self.registry.get("/help")
        assert cmd is not None
        assert cmd.name == "/help"

    def test_get_returns_none_for_unknown(self) -> None:
        cmd = self.registry.get("/nonexistent")
        assert cmd is None

    def test_has_returns_true_for_known_commands(self) -> None:
        assert self.registry.has("/help") is True
        assert self.registry.has("/model") is True

    def test_has_returns_false_for_unknown(self) -> None:
        assert self.registry.has("/nonexistent") is False

    def test_has_rpc_mapping_for_rpc_commands(self) -> None:
        assert self.registry.has_rpc_mapping("/model") is True
        assert self.registry.has_rpc_mapping("/compact") is True
        assert self.registry.has_rpc_mapping("/agent") is True

    def test_has_rpc_mapping_false_for_passthrough(self) -> None:
        assert self.registry.has_rpc_mapping("/help") is False
        assert self.registry.has_rpc_mapping("/nonexistent") is False

    def test_parse_command_line_basic(self) -> None:
        result = self.registry.parse_command_line("/help")
        assert result is not None
        assert result.name == "/help"
        assert result.args == []

    def test_parse_command_line_with_args(self) -> None:
        result = self.registry.parse_command_line("/model gpt-4")
        assert result is not None
        assert result.name == "/model"
        assert result.args == ["gpt-4"]

    def test_parse_command_line_multiple_args(self) -> None:
        result = self.registry.parse_command_line("/agent select my-agent")
        assert result is not None
        assert result.name == "/agent"
        assert result.args == ["select", "my-agent"]

    def test_parse_command_line_with_whitespace(self) -> None:
        result = self.registry.parse_command_line("  /help  ")
        assert result is not None
        assert result.name == "/help"
        assert result.args == []

    def test_parse_command_line_non_command(self) -> None:
        assert self.registry.parse_command_line("hello world") is None
        assert self.registry.parse_command_line("") is None
