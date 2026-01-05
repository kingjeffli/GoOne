Param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$Args
)

$ErrorActionPreference = "Stop"

function Resolve-GoExe {
  $candidates = @()
  try {
    $cmd = Get-Command go -ErrorAction Stop
    if ($cmd.Source) { $candidates += $cmd.Source }
  } catch {}

  if ($env:GOROOT) {
    $candidates += (Join-Path $env:GOROOT "bin\go.exe")
  }
  $candidates += @(
    "C:\Go\bin\go.exe",
    "C:\Program Files\Go\bin\go.exe"
  )

  foreach ($p in $candidates | Select-Object -Unique) {
    if ($p -and (Test-Path $p)) { return $p }
  }
  return $null
}

Push-Location (Split-Path $PSScriptRoot -Parent)
try {
  $goExe = Resolve-GoExe
  if (-not $goExe) {
    Write-Host "[gencmdproto] ERROR: Go toolchain not found (go.exe)." -ForegroundColor Red
    Write-Host "[gencmdproto] Install Go for Windows, or run the WSL script: ./scripts/gencmdproto.sh" -ForegroundColor Yellow
    throw "go.exe not found"
  }

  & $goExe run .\tools\cmd\gencmdproto @Args
} finally {
  Pop-Location
}


