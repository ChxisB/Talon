#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Spectre Proxy management script for Windows (PowerShell).
.DESCRIPTION
    Build, start, stop, and manage the Spectre Proxy Docker services.
    Drop-in equivalent of docker/run.sh for Windows users.
.EXAMPLE
    .\docker\run.ps1 up        # Build and start all services
    .\docker\run.ps1 down      # Stop all services
    .\docker\run.ps1 logs      # Follow logs
    .\docker\run.ps1 dashboard # Open dashboard in browser
#>

param(
    [Parameter(Position = 0)]
    [string]$Command = "help"
)

$ComposeFile = Join-Path $PSScriptRoot "docker-compose.yml"

# ─── Config ──────────────────────────────────────────────────────────
# Source API keys from ~/.spectre-proxy/.env if it exists
$EnvFile = Join-Path $HOME ".spectre-proxy" ".env"
if (Test-Path $EnvFile) {
    Get-Content $EnvFile | ForEach-Object {
        if ($_ -match "^\s*([^#=]+)=(.*)\s*$") {
            $key = $matches[1].Trim()
            $val = $matches[2].Trim()
            Set-Item -Path "Env:$key" -Value $val -ErrorAction SilentlyContinue
        }
    }
}

# Default model
if (-not $env:MODEL) {
    $env:MODEL = "openrouter/anthropic/claude-sonnet-4"
}

# ─── Help ────────────────────────────────────────────────────────────
function Show-Help {
    @"
Usage: .\docker\run.ps1 [command]

Commands:
  up        Build and start all services    (docker compose up --build -d)
  down      Stop all services               (docker compose down)
  logs      Follow logs                     (docker compose logs -f)
  restart   Restart all services
  status    Show container status
  build     Rebuild images without starting
  shell     Open a shell in the agent container
  dashboard Open the dashboard in your browser
"@
    exit 0
}

# ─── Commands ────────────────────────────────────────────────────────
switch ($Command) {
    "up" {
        Write-Host "🚀 Building and starting Spectre Proxy..." -ForegroundColor Cyan
        docker compose -f $ComposeFile up --build -d
        Write-Host ""
        Write-Host "  Dashboard:  http://localhost:3000" -ForegroundColor Green
        Write-Host "  Proxy:      http://localhost:8082" -ForegroundColor Green
        Write-Host "  Proxy API:  curl http://localhost:8082/v1/models" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Set API keys: `$env:OPENROUTER_API_KEY = 'sk-...'" -ForegroundColor Yellow
        Write-Host "  Or add them to: $EnvFile" -ForegroundColor Yellow
    }
    "down" {
        Write-Host "🛑 Stopping Spectre Proxy..." -ForegroundColor Cyan
        docker compose -f $ComposeFile down
    }
    "logs" {
        docker compose -f $ComposeFile logs -f
    }
    "restart" {
        Write-Host "🔄 Restarting..." -ForegroundColor Cyan
        docker compose -f $ComposeFile down
        docker compose -f $ComposeFile up --build -d
    }
    "status" {
        docker compose -f $ComposeFile ps
    }
    "build" {
        Write-Host "🔨 Building images..." -ForegroundColor Cyan
        docker compose -f $ComposeFile build
    }
    "shell" {
        Write-Host "🐚 Opening shell in proxy container..." -ForegroundColor Cyan
        docker exec -it spectre-proxy sh
    }
    "dashboard" {
        Write-Host "📂 Opening dashboard..." -ForegroundColor Cyan
        Start-Process "http://localhost:3000"
    }
    default {
        Show-Help
    }
}
