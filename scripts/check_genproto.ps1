# Regenerate api/gen and fail if the working tree still differs (same contract as ./main.sh check-genproto).
# Requires: go, git, protoc (via tools/cmd/genproto discovery).

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

$goExe = Get-Command go -ErrorAction Stop | Select-Object -ExpandProperty Source
$module = (Select-String -Path (Join-Path $repoRoot "go.mod") -Pattern '^module\s+(\S+)' | ForEach-Object { $_.Matches.Groups[1].Value } | Select-Object -First 1)
if (-not $module) { throw "Cannot read module path from go.mod" }

Write-Host "[check_genproto] go run tools/cmd/genproto module=$module"
& $goExe run .\tools\cmd\genproto -module $module -out . -proto_root api/proto
if ($LASTEXITCODE -ne 0) { throw "genproto failed with exit code $LASTEXITCODE" }

Write-Host "[check_genproto] git diff --quiet api/gen"
git -C $repoRoot diff --quiet -- api/gen
if ($LASTEXITCODE -ne 0) {
  throw "api/gen is out of date. Run scripts/proto_goone.ps1 or genproto, then commit."
}

Write-Host "[check_genproto] OK"
