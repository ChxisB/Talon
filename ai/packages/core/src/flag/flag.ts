import { Config } from "effect"

export function truthy(key: string) {
  const value = process.env[key]?.toLowerCase()
  return value === "true" || value === "1"
}

const copy = process.env["TALON_EXPERIMENTAL_DISABLE_COPY_ON_SELECT"]
const fff = process.env["TALON_DISABLE_FFF"]

function enabledByExperimental(key: string) {
  return process.env[key] === undefined ? truthy("TALON_EXPERIMENTAL") : truthy(key)
}

export const Flag = {
  OTEL_EXPORTER_OTLP_ENDPOINT: process.env["OTEL_EXPORTER_OTLP_ENDPOINT"],
  OTEL_EXPORTER_OTLP_HEADERS: process.env["OTEL_EXPORTER_OTLP_HEADERS"],

  TALON_AUTO_HEAP_SNAPSHOT: truthy("TALON_AUTO_HEAP_SNAPSHOT"),
  TALON_GIT_BASH_PATH: process.env["TALON_GIT_BASH_PATH"],
  TALON_CONFIG: process.env["TALON_CONFIG"],
  TALON_CONFIG_CONTENT: process.env["TALON_CONFIG_CONTENT"],
  TALON_DISABLE_AUTOUPDATE: truthy("TALON_DISABLE_AUTOUPDATE"),
  TALON_ALWAYS_NOTIFY_UPDATE: truthy("TALON_ALWAYS_NOTIFY_UPDATE"),
  TALON_DISABLE_PRUNE: truthy("TALON_DISABLE_PRUNE"),
  TALON_DISABLE_TERMINAL_TITLE: truthy("TALON_DISABLE_TERMINAL_TITLE"),
  TALON_SHOW_TTFD: truthy("TALON_SHOW_TTFD"),
  TALON_DISABLE_AUTOCOMPACT: truthy("TALON_DISABLE_AUTOCOMPACT"),
  TALON_DISABLE_MODELS_FETCH: truthy("TALON_DISABLE_MODELS_FETCH"),
  TALON_DISABLE_MOUSE: truthy("TALON_DISABLE_MOUSE"),
  TALON_FAKE_VCS: process.env["TALON_FAKE_VCS"],
  TALON_SERVER_PASSWORD: process.env["TALON_SERVER_PASSWORD"],
  TALON_SERVER_USERNAME: process.env["TALON_SERVER_USERNAME"],
  TALON_DISABLE_FFF: fff === undefined ? process.platform === "win32" : truthy("TALON_DISABLE_FFF"),

  // Experimental
  TALON_EXPERIMENTAL_FILEWATCHER: Config.boolean("TALON_EXPERIMENTAL_FILEWATCHER").pipe(
    Config.withDefault(false),
  ),
  TALON_EXPERIMENTAL_DISABLE_FILEWATCHER: Config.boolean("TALON_EXPERIMENTAL_DISABLE_FILEWATCHER").pipe(
    Config.withDefault(false),
  ),
  TALON_EXPERIMENTAL_DISABLE_COPY_ON_SELECT:
    copy === undefined ? process.platform === "win32" : truthy("TALON_EXPERIMENTAL_DISABLE_COPY_ON_SELECT"),
  TALON_MODELS_URL: process.env["TALON_MODELS_URL"],
  TALON_MODELS_PATH: process.env["TALON_MODELS_PATH"],
  TALON_DB: process.env["TALON_DB"],

  TALON_WORKSPACE_ID: process.env["TALON_WORKSPACE_ID"],
  TALON_EXPERIMENTAL_WORKSPACES: enabledByExperimental("TALON_EXPERIMENTAL_WORKSPACES"),

  // Evaluated at access time (not module load) because tests, the CLI, and
  // external tooling set these env vars at runtime.
  get TALON_DISABLE_PROJECT_CONFIG() {
    return truthy("TALON_DISABLE_PROJECT_CONFIG")
  },
  get TALON_EXPERIMENTAL_REFERENCES() {
    return enabledByExperimental("TALON_EXPERIMENTAL_REFERENCES")
  },
  get TALON_TUI_CONFIG() {
    return process.env["TALON_TUI_CONFIG"]
  },
  get TALON_CONFIG_DIR() {
    return process.env["TALON_CONFIG_DIR"]
  },
  get TALON_PURE() {
    return truthy("TALON_PURE")
  },
  get TALON_PERMISSION() {
    return process.env["TALON_PERMISSION"]
  },
  get TALON_PLUGIN_META_FILE() {
    return process.env["TALON_PLUGIN_META_FILE"]
  },
  get TALON_CLIENT() {
    return process.env["TALON_CLIENT"] ?? "cli"
  },
}
