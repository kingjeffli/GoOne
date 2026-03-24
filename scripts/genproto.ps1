Param(
  [string]$Module = "github.com/Iori372552686/GoOne"
)

$ErrorActionPreference = "Stop"

Write-Host "[genproto.ps1] Module=$Module"

try {
  $null = Get-Command protoc -ErrorAction Stop
} catch {
  Write-Host "[genproto.ps1] protoc not found on PATH."
  Write-Host "[genproto.ps1] Recommendation: run in WSL: ./scripts/genproto.sh"
  throw
}

Push-Location (Split-Path $PSScriptRoot -Parent)
try {
  go run .\tools\cmd\genproto -module $Module
} finally {
  Pop-Location
}


