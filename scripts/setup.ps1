$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repositoryRoot

$environmentFiles = @{
    ".env.root" = ".env.root.example"
    ".env.be" = ".env.be.example"
    ".env.fe" = ".env.fe.example"
    ".env.worker" = ".env.worker.example"
}

foreach ($target in $environmentFiles.Keys) {
    if (-not (Test-Path -LiteralPath $target)) {
        Copy-Item -LiteralPath $environmentFiles[$target] -Destination $target
        Write-Host "Created $target"
    } else {
        Write-Host "Keeping existing $target"
    }
}

Write-Host "Installing JavaScript dependencies..."
corepack pnpm install

Write-Host "Downloading Go dependencies..."
go -C backend mod download

$venvPython = Join-Path $repositoryRoot "worker\.venv\Scripts\python.exe"
if (-not (Test-Path -LiteralPath $venvPython)) {
    Write-Host "Creating Python virtual environment..."
    python -m venv worker\.venv
}

Write-Host "Installing Python dependencies..."
& $venvPython -m pip install --upgrade pip
& $venvPython -m pip install -e "./worker[dev]"

Write-Host ""
Write-Host "Setup complete."
Write-Host "The pnpm worker commands use worker\.venv automatically."
Write-Host "Start infrastructure and apps:"
Write-Host "  pnpm dev:all"
