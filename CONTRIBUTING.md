# Contributing

Thanks for your interest in contributing to Talon!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/Talon.git`
3. Install dependencies: `bash scripts/install.sh`
4. Create a branch: `git checkout -b my-feature`

## Development

```bash
# Build and run
bash scripts/install.sh --dev

# Run the Go backend
cd backend && go run ./cmd/server/

# Run the TUI from source
cd ai/packages/talon && bun run src/index.ts
```

## Guidelines

- Keep changes focused and well-documented
- Test your changes before submitting
- Follow the existing code style
- One pull request per feature/fix

## Code Style

- **TypeScript**: camelCase variables, PascalCase types, no semicolons
- **Go**: standard Go formatting (`gofmt`)
- **Zig**: existing project conventions

## Pull Request Process

1. Update the README.md if needed
2. Ensure the build passes: `bash scripts/install.sh`
3. Submit a PR with a clear description of changes
