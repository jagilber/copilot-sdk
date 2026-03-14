/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *--------------------------------------------------------------------------------------------*/

using GitHub.Copilot.SDK;
using Xunit;

namespace GitHub.Copilot.SDK.Test;

public class CommandsTests
{
    [Fact]
    public void BuiltinCommands_ContainsExpectedCommands()
    {
        var registry = new CommandRegistry();
        var commands = CommandRegistry.List();
        var names = commands.Select(c => c.Name).ToList();

        Assert.Contains("/help", names);
        Assert.Contains("/model", names);
        Assert.Contains("/compact", names);
        Assert.Contains("/clear", names);
        Assert.Contains("/agent", names);

        foreach (var cmd in commands)
        {
            Assert.StartsWith("/", cmd.Name);
            Assert.NotEmpty(cmd.Description);
        }
    }

    [Fact]
    public void Get_ReturnsCommandByName()
    {
        var registry = new CommandRegistry();
        var cmd = registry.Get("/help");

        Assert.NotNull(cmd);
        Assert.Equal("/help", cmd.Name);
    }

    [Fact]
    public void Get_ReturnsNullForUnknown()
    {
        var registry = new CommandRegistry();
        var cmd = registry.Get("/nonexistent");

        Assert.Null(cmd);
    }

    [Fact]
    public void Has_ReturnsTrueForKnown()
    {
        var registry = new CommandRegistry();

        Assert.True(registry.Has("/help"));
        Assert.True(registry.Has("/model"));
    }

    [Fact]
    public void Has_ReturnsFalseForUnknown()
    {
        var registry = new CommandRegistry();

        Assert.False(registry.Has("/nonexistent"));
    }

    [Fact]
    public void HasRpcMapping_ForRpcCommands()
    {
        var registry = new CommandRegistry();

        Assert.True(CommandRegistry.HasRpcMapping("/model"));
        Assert.True(CommandRegistry.HasRpcMapping("/compact"));
        Assert.True(CommandRegistry.HasRpcMapping("/agent"));
    }

    [Fact]
    public void HasRpcMapping_FalseForPassthrough()
    {
        var registry = new CommandRegistry();

        Assert.False(CommandRegistry.HasRpcMapping("/help"));
        Assert.False(CommandRegistry.HasRpcMapping("/nonexistent"));
    }

    [Fact]
    public void ParseCommandLine_Basic()
    {
        var registry = new CommandRegistry();
        var result = CommandRegistry.ParseCommandLine("/help");

        Assert.NotNull(result);
        Assert.Equal("/help", result.Value.Name);
        Assert.Empty(result.Value.Args);
    }

    [Fact]
    public void ParseCommandLine_WithArgs()
    {
        var registry = new CommandRegistry();
        var result = CommandRegistry.ParseCommandLine("/model gpt-4");

        Assert.NotNull(result);
        Assert.Equal("/model", result.Value.Name);
        Assert.Single(result.Value.Args);
        Assert.Equal("gpt-4", result.Value.Args[0]);
    }

    [Fact]
    public void ParseCommandLine_MultipleArgs()
    {
        var registry = new CommandRegistry();
        var result = CommandRegistry.ParseCommandLine("/agent select my-agent");

        Assert.NotNull(result);
        Assert.Equal("/agent", result.Value.Name);
        Assert.Equal(2, result.Value.Args.Length);
        Assert.Equal("select", result.Value.Args[0]);
        Assert.Equal("my-agent", result.Value.Args[1]);
    }

    [Fact]
    public void ParseCommandLine_WithWhitespace()
    {
        var registry = new CommandRegistry();
        var result = CommandRegistry.ParseCommandLine("  /help  ");

        Assert.NotNull(result);
        Assert.Equal("/help", result.Value.Name);
    }

    [Fact]
    public void ParseCommandLine_NonCommand_ReturnsNull()
    {
        var registry = new CommandRegistry();

        Assert.Null(CommandRegistry.ParseCommandLine("hello world"));
        Assert.Null(CommandRegistry.ParseCommandLine(""));
    }
}
