import * as vscode from "vscode";
import { spawn, ChildProcess } from "child_process";
import { existsSync } from "fs";
import { homedir } from "os";
import { join } from "path";
import { SpectreChatViewProvider } from "./panel";

let proxyProcess: ChildProcess | null = null;
let statusBarItem: vscode.StatusBarItem;
let outputChannel: vscode.OutputChannel;
let chatProvider: SpectreChatViewProvider | null = null;

const PROJECT_DIR = join(homedir(), "Spectre CC", "spectre-proxy");

function getConfig() {
  const config = vscode.workspace.getConfiguration("spectre-proxy");
  return {
    proxyPort: config.get<number>("proxyPort", 8082),
    dashboardUrl: config.get<string>("dashboardUrl", "http://localhost:3000"),
    proxyApiKey: config.get<string>("proxyApiKey", "spectre-proxy"),
    proxyBinPath: config.get<string>("proxyBinPath", ""),
    autoStart: config.get<boolean>("autoStart", false),
  };
}

export function activate(context: vscode.ExtensionContext) {
  outputChannel = vscode.window.createOutputChannel("Spectre Proxy");

  // Status bar item
  statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
  statusBarItem.command = "spectre-proxy.dashboard";
  statusBarItem.tooltip = "Spectre Proxy — Click to open dashboard";
  context.subscriptions.push(statusBarItem);
  statusBarItem.show();

  // Register commands
  context.subscriptions.push(
    vscode.commands.registerCommand("spectre-proxy.dashboard", openDashboard),
    vscode.commands.registerCommand("spectre-proxy.startProxy", startProxy),
    vscode.commands.registerCommand("spectre-proxy.stopProxy", stopProxy),
    vscode.commands.registerCommand("spectre-proxy.restartProxy", restartProxy),
    vscode.commands.registerCommand("spectre-proxy.openSettings", openSettings),
    vscode.commands.registerCommand("spectre-proxy.terminal", openTerminal),
  );

  checkProxyStatus();
  setInterval(checkProxyStatus, 10000);

  // Auto-start if configured
  const config = getConfig();
  if (config.autoStart) startProxy();

  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("spectre-proxy")) {
        checkProxyStatus();
        // Notify webview of config change
        if (chatProvider) {
          chatProvider.updateConfig(getConfig());
        }
      }
    })
  );

  // Register sidebar view (now with splash + status)
  chatProvider = new SpectreChatViewProvider(context.extensionUri, {
    checkProxyStatus,
    startProxy,
    stopProxy,
    openDashboard,
    getConfig,
  });
  context.subscriptions.push(
    vscode.window.registerWebviewViewProvider("spectre-proxy.chatView", chatProvider, {
      webviewOptions: { retainContextWhenHidden: true },
    })
  );
}

export function deactivate() {
  stopProxy();
}

// ─── Status Bar ──────────────────────────────────────────────────

async function checkProxyStatus(): Promise<boolean> {
  const config = getConfig();
  try {
    const resp = await fetch(`http://127.0.0.1:${config.proxyPort}/health`, { signal: AbortSignal.timeout(2000) });
    if (resp.ok) {
      statusBarItem.text = "$(pass) Spectre Proxy";
      statusBarItem.backgroundColor = undefined;
      statusBarItem.color = undefined;
      return true;
    }
  } catch {}
  if (proxyProcess) {
    statusBarItem.text = "$(loading~spin) Spectre Proxy (starting…)";
  } else {
    statusBarItem.text = "$(circle-slash) Spectre Proxy";
    statusBarItem.backgroundColor = new vscode.ThemeColor("statusBarItem.warningBackground");
  }
  return false;
}

// ─── Commands ────────────────────────────────────────────────────

async function openDashboard() {
  const config = getConfig();
  vscode.env.openExternal(vscode.Uri.parse(config.dashboardUrl));
}

async function startProxy() {
  if (proxyProcess) {
    vscode.window.showInformationMessage("Spectre proxy is already running");
    return;
  }

  const config = getConfig();
  const binPath = findProxyBinary();
  if (!binPath) {
    const action = await vscode.window.showErrorMessage(
      "Spectre proxy binary not found. Build it first.",
      "Build Now",
      "Open Settings"
    );
    if (action === "Build Now") await buildProxy();
    else if (action === "Open Settings") openSettings();
    return;
  }

  outputChannel.appendLine(`Starting Spectre proxy: ${binPath}`);
  checkProxyStatus();

  proxyProcess = spawn(binPath, [], {
    stdio: ["ignore", "pipe", "pipe"],
    env: { ...process.env },
  });

  proxyProcess.stdout?.on("data", (data) => outputChannel.append(data.toString()));
  proxyProcess.stderr?.on("data", (data) => outputChannel.append(data.toString()));
  proxyProcess.on("exit", (code) => {
    outputChannel.appendLine(`Proxy exited with code ${code}`);
    proxyProcess = null;
    checkProxyStatus();
  });

  // Wait for ready (up to 10s)
  for (let i = 0; i < 20; i++) {
    await sleep(500);
    try {
      const resp = await fetch(`http://127.0.0.1:${config.proxyPort}/health`);
      if (resp.ok) {
        vscode.window.showInformationMessage("Spectre proxy started");
        checkProxyStatus();
        return;
      }
    } catch {}
  }

  vscode.window.showWarningMessage("Spectre proxy started but not yet responding");
  checkProxyStatus();
}

async function stopProxy() {
  if (!proxyProcess) {
    vscode.window.showInformationMessage("Spectre proxy is not running");
    return;
  }
  proxyProcess.kill("SIGTERM");
  proxyProcess = null;
  outputChannel.appendLine("Proxy stopped");
  checkProxyStatus();
}

async function restartProxy() {
  await stopProxy();
  await sleep(500);
  await startProxy();
}

async function openSettings() {
  vscode.commands.executeCommand("workbench.action.openSettings", "@ext:spectre-proxy");
}

function openTerminal() {
  const terminal = vscode.window.createTerminal({ name: "Spectre Proxy" });
  terminal.show();
  terminal.sendText(`cd "${PROJECT_DIR}"`);
}

// ─── Helpers ────────────────────────────────────────────────────

/** Returns the proxy binary filename for the current platform. */
function proxyBinaryName(): string {
  return process.platform === "win32" ? "spectre-server.exe" : "spectre-server";
}

function findProxyBinary(): string | null {
  const config = getConfig();
  if (config.proxyBinPath && existsSync(config.proxyBinPath)) return config.proxyBinPath;

  const bin = proxyBinaryName();
  const candidates = [
    join(PROJECT_DIR, "agent", bin),
    join(PROJECT_DIR, "agent", "cmd", "spectre-server", bin),
    join(PROJECT_DIR, bin),
    join(homedir(), "go", "bin", bin),
    bin,
  ];

  for (const c of candidates) {
    if (existsSync(c)) return c;
  }
  return null;
}

async function buildProxy(): Promise<boolean> {
  const agentDir = join(PROJECT_DIR, "agent");
  if (!existsSync(join(agentDir, "go.mod"))) {
    vscode.window.showErrorMessage("Agent directory not found at " + agentDir);
    return false;
  }
  const terminal = vscode.window.createTerminal({ name: "Build Spectre Proxy" });
  terminal.show();
  terminal.sendText(`cd "${agentDir}" && go build -o ${proxyBinaryName()} ./cmd/spectre-server/`);
  return true;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
