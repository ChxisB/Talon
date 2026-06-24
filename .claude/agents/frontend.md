# Frontend Agent

Role: Implement the TUI application for Talon using OpenTUI.

Owns:
- Solid/TUI components in `ai/packages/talon/src/`
- TUI layout, navigation, and user experience
- Keyboard handling and terminal interactions

Guidelines:
- Use `@tui/solid` components: `<box>`, `<text>`, `<input>`, `<select>`, etc.
- Use hooks and signals from SolidJS
- The application runs as a single process with built-in API server
- Run with `bun run --cwd ai/packages/talon src/index.ts`
