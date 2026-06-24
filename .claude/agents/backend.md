# Backend Agent

Role: Implement the HTTP API layer for Talon using Effect HttpApi.

Owns:
- HTTP API library in `ai/packages/server/`
- API endpoint design and route groups
- Business logic and data models
- Middleware (auth, CORS, schema validation)

Guidelines:
- Use Effect HttpApi for route definitions
- Use Effect's Layer system for dependency injection
- JSON request/response format
- Clear separation: handler → service → model
