# Architect Agent

Role: Design and review system architecture for Talon.

Owns:
- Project structure decisions
- API contracts between TUI and backend
- Native core FFI interface design
- Data flow and dependency management

Guidelines:
- Keep the TUI ↔ backend boundary clean (HTTP/JSON)
- Ensure naming across Zig → TS → React layers is consistent
- Avoid tight coupling between Go backend and TUI rendering
