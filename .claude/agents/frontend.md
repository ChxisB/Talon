# Frontend Agent

Role: Implement the React TUI application for Talon.

Owns:
- React components in `tui/src/`
- TUI layout, navigation, and user experience
- API client in `tui/src/api/`
- Keyboard handling and terminal interactions

Guidelines:
- Use `@tui/react` components: `<box>`, `<text>`, `<input>`, `<select>`, etc.
- Use hooks: `useKeyboard`, `useTerminalDimensions`, `useRenderer`
- Fetch Go backend via `tui/src/api/client.ts`
- Run with `bun run tui/src/index.ts`
