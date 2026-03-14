/* eslint-disable @typescript-eslint/no-explicit-any */
import { describe, expect, it, onTestFinished, vi, beforeEach } from "vitest";
import { approveAll, CopilotClient } from "../src/index.js";
import { CommandRegistry, BUILTIN_COMMANDS } from "../src/commands.js";

describe("SlashCommand Types", () => {
    it("BUILTIN_COMMANDS contains expected commands", () => {
        const commandNames = BUILTIN_COMMANDS.map((c) => c.name);

        expect(commandNames).toContain("/help");
        expect(commandNames).toContain("/model");
        expect(commandNames).toContain("/compact");
        expect(commandNames).toContain("/clear");
        expect(commandNames).toContain("/agent");

        for (const cmd of BUILTIN_COMMANDS) {
            expect(cmd.name).toMatch(/^\//);
            expect(cmd.description).toBeTruthy();
            expect(typeof cmd.description).toBe("string");
        }
    });

    it("each command has required SlashCommand shape", () => {
        for (const cmd of BUILTIN_COMMANDS) {
            expect(cmd).toHaveProperty("name");
            expect(cmd).toHaveProperty("description");
            // parameters is optional
            if (cmd.parameters) {
                expect(Array.isArray(cmd.parameters)).toBe(true);
                for (const param of cmd.parameters) {
                    expect(param).toHaveProperty("name");
                    expect(param).toHaveProperty("description");
                    expect(param).toHaveProperty("required");
                }
            }
        }
    });
});

describe("CommandRegistry", () => {
    let registry: CommandRegistry;

    beforeEach(() => {
        registry = new CommandRegistry();
    });

    it("list() returns all builtin commands", () => {
        const commands = registry.list();
        expect(commands.length).toBeGreaterThan(0);
        expect(commands.length).toBe(BUILTIN_COMMANDS.length);
    });

    it("get() returns a command by name", () => {
        const cmd = registry.get("/help");
        expect(cmd).toBeDefined();
        expect(cmd!.name).toBe("/help");
    });

    it("get() returns undefined for unknown command", () => {
        const cmd = registry.get("/nonexistent");
        expect(cmd).toBeUndefined();
    });

    it("has() returns true for known commands", () => {
        expect(registry.has("/help")).toBe(true);
        expect(registry.has("/model")).toBe(true);
    });

    it("has() returns false for unknown commands", () => {
        expect(registry.has("/nonexistent")).toBe(false);
    });

    it("hasRpcMapping() identifies commands with RPC equivalents", () => {
        expect(registry.hasRpcMapping("/model")).toBe(true);
        expect(registry.hasRpcMapping("/compact")).toBe(true);
        expect(registry.hasRpcMapping("/agent")).toBe(true);
    });

    it("hasRpcMapping() returns false for passthrough commands", () => {
        expect(registry.hasRpcMapping("/help")).toBe(false);
        expect(registry.hasRpcMapping("/nonexistent")).toBe(false);
    });

    it("parseCommandLine() parses command name and args", () => {
        expect(registry.parseCommandLine("/help")).toEqual({
            name: "/help",
            args: [],
        });
        expect(registry.parseCommandLine("/model gpt-4")).toEqual({
            name: "/model",
            args: ["gpt-4"],
        });
        expect(registry.parseCommandLine("/agent select my-agent")).toEqual({
            name: "/agent",
            args: ["select", "my-agent"],
        });
    });

    it("parseCommandLine() handles leading/trailing whitespace", () => {
        expect(registry.parseCommandLine("  /help  ")).toEqual({
            name: "/help",
            args: [],
        });
    });

    it("parseCommandLine() returns null for non-command text", () => {
        expect(registry.parseCommandLine("hello world")).toBeNull();
        expect(registry.parseCommandLine("")).toBeNull();
    });
});

describe("CopilotSession slash commands", () => {
    it("supportedCommands() returns list of slash commands", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const commands = session.supportedCommands();
        expect(commands.length).toBeGreaterThan(0);
        for (const cmd of commands) {
            expect(cmd.name).toMatch(/^\//);
            expect(cmd.description).toBeTruthy();
        }
    });

    it("sendCommand() calls RPC for /model command", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const spy = vi.spyOn(session.rpc.model, "switchTo");
        spy.mockResolvedValue({} as any);

        const result = await session.sendCommand("/model", ["gpt-4"]);
        expect(result.handled).toBe(true);
        expect(result.method).toBe("rpc");
        expect(spy).toHaveBeenCalledWith(expect.objectContaining({ modelId: "gpt-4" }));
    });

    it("sendCommand() calls RPC for /compact command", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const spy = vi.spyOn(session.rpc.compaction, "compact");
        spy.mockResolvedValue({} as any);

        const result = await session.sendCommand("/compact");
        expect(result.handled).toBe(true);
        expect(result.method).toBe("rpc");
        expect(spy).toHaveBeenCalled();
    });

    it("sendCommand() calls RPC for /agent select", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const spy = vi.spyOn(session.rpc.agent, "select");
        spy.mockResolvedValue({} as any);

        const result = await session.sendCommand("/agent", ["select", "my-agent"]);
        expect(result.handled).toBe(true);
        expect(result.method).toBe("rpc");
        expect(spy).toHaveBeenCalledWith(expect.objectContaining({ name: "my-agent" }));
    });

    it("sendCommand() falls back to send() for unknown commands", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const spy = vi.spyOn(session, "send");
        spy.mockResolvedValue("msg-123");

        const result = await session.sendCommand("/some-unknown-command", ["arg1"]);
        expect(result.handled).toBe(true);
        expect(result.method).toBe("passthrough");
        expect(spy).toHaveBeenCalledWith(
            expect.objectContaining({ prompt: "/some-unknown-command arg1" })
        );
    });

    it("sendCommand() returns help text for /help", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const result = await session.sendCommand("/help");
        expect(result.handled).toBe(true);
        expect(result.method).toBe("local");
        expect(result.output).toBeTruthy();
        expect(result.output).toContain("/help");
        expect(result.output).toContain("/model");
    });

    it("sendCommand() handles /clear via local handler", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const result = await session.sendCommand("/clear");
        expect(result.handled).toBe(true);
        expect(result.method).toBe("local");
    });

    it("sendCommand() parses full command string", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        const spy = vi.spyOn(session.rpc.model, "switchTo");
        spy.mockResolvedValue({} as any);

        // Can also pass the full command string
        const result = await session.sendCommand("/model gpt-4");
        expect(result.handled).toBe(true);
        expect(spy).toHaveBeenCalledWith(expect.objectContaining({ modelId: "gpt-4" }));
    });

    it("sendCommand() throws for empty command", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        await expect(session.sendCommand("")).rejects.toThrow(/invalid.*command/i);
    });

    it("sendCommand() throws for non-slash-prefixed input", async () => {
        const client = new CopilotClient();
        await client.start();
        onTestFinished(() => client.forceStop());

        const session = await client.createSession({
            onPermissionRequest: approveAll,
        });

        await expect(session.sendCommand("hello")).rejects.toThrow(/must start with/i);
    });
});
