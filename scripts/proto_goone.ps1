Param(
  [string]$Module = "github.com/Iori372552686/GoOne",
  [string]$OutDir = ".",
  [string]$ProtoRoot = "api/proto",
  [string]$BinDir = ".bin",
  [switch]$UseWSL
)

$ErrorActionPreference = "Stop"

function Resolve-GoExe {
  $candidates = @()
  try {
    $cmd = Get-Command go -ErrorAction Stop
    if ($cmd.Source) { $candidates += $cmd.Source }
  }
  catch {}

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

function Resolve-ProtocExe([string]$repoRoot) {
  try {
    $cmd = Get-Command protoc -ErrorAction Stop
    if ($cmd.Source) { return $cmd.Source }
  }
  catch {}

  $vendored = Join-Path $repoRoot "lib\contrib\protoc\protoc-33.2-win64\bin\protoc.exe"
  if (Test-Path $vendored) { return $vendored }

  return $null
}

function Invoke-InWSL([string]$repoRoot) {
  $wsl = Get-Command wsl -ErrorAction SilentlyContinue
  if (-not $wsl) {
    throw "wsl.exe not found; can't use WSL fallback."
  }
  # Use -e to avoid shell-escaping issues with backslashes in Windows paths.
  $wslPath = & wsl.exe -e wslpath -a $repoRoot
  if (-not $wslPath) {
    throw "failed to convert repo path to WSL path: $repoRoot"
  }
  Write-Host "[proto_goone] running in WSL: cd $wslPath && ./scripts/proto_goone.sh"
  & wsl.exe -e bash -lc "cd `"$wslPath`" && ./scripts/proto_goone.sh"
}

Write-Host "[proto_goone] build plugins -> $BinDir"
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$goExe = Resolve-GoExe
if (-not $goExe) {
  if ($UseWSL -or ($env:PROTO_GOONE_USE_WSL -eq "1")) {
    Invoke-InWSL $repoRoot
    exit 0
  }

  Write-Host "[proto_goone] ERROR: Go toolchain not found (go.exe)." -ForegroundColor Red
  Write-Host "[proto_goone] Fix: install Go for Windows, OR run with WSL fallback:" -ForegroundColor Yellow
  Write-Host "  .\scripts\proto_goone.ps1 -UseWSL" -ForegroundColor Yellow
  throw "go.exe not found"
}


$cmdProto = Join-Path $ProtoRoot "goone/cmd/v1/cmd.proto"
if (($env:GEN_CMD_PROTO -eq "1") -or (-not (Test-Path $cmdProto))) {
  Write-Host "[proto_goone] ensure cmd.proto -> $cmdProto"
  & (Join-Path $PSScriptRoot "gencmdproto.ps1")
}

& $goExe build -o (Join-Path $BinDir "protoc-gen-goone.exe") .\tools\protoc-gen-goone
& $goExe build -o (Join-Path $BinDir "protoc-gen-go.exe") "google.golang.org/protobuf/cmd/protoc-gen-go"

$protoc = Resolve-ProtocExe $repoRoot
if (-not $protoc) {
  Write-Host "[proto_goone] ERROR: protoc not found." -ForegroundColor Red
  Write-Host "[proto_goone] Fix: install protoc on PATH, OR use repo-vendored protoc at lib/contrib/protoc/protoc-33.2-win64/bin/protoc.exe." -ForegroundColor Yellow
  Write-Host "[proto_goone] If you prefer WSL, run: .\scripts\proto_goone.ps1 -UseWSL" -ForegroundColor Yellow
  throw "protoc not found"
}

$include = @($ProtoRoot)

# Add vendored include dirs if present (mainly for CI / portable setups)
$cand = @(
  "lib/contrib/protoc/protoc-33.2-win64/include",
  "lib/util/deps/protoc/protoc-3.11.4-win64/include"
)
foreach ($c in $cand) {
  if (Test-Path $c) { $include += $c }
}

Write-Host "[proto_goone] collect proto inputs (exclude api/proto/third_party)"
$protos = @()
$roots = @(
  Join-Path $ProtoRoot "goone",
  Join-Path $ProtoRoot "game"
)
foreach ($r in $roots) {
  if (-not (Test-Path $r)) { continue }
  $protos += Get-ChildItem -Recurse -File -Path $r -Filter *.proto | ForEach-Object {
    # make path relative to ProtoRoot for protoc import mapping
    $_.FullName.Substring((Resolve-Path $ProtoRoot).Path.Length + 1).Replace("\", "/")
  }
}
$protos = $protos | Sort-Object

Write-Host "[proto_goone] protoc=$protoc"
Write-Host "[proto_goone] module=$Module"
Write-Host ("[proto_goone] inputs={0}" -f $protos.Count)

$protocArgs = @()
foreach ($i in $include) { $protocArgs += @("-I", $i) }

$protocArgs += @("--plugin=protoc-gen-go=$(Resolve-Path (Join-Path $BinDir "protoc-gen-go.exe"))")
$protocArgs += @("--plugin=protoc-gen-goone=$(Resolve-Path (Join-Path $BinDir "protoc-gen-goone.exe"))")
$protocArgs += @("--go_out=$OutDir", "--go_opt=module=$Module", "--go_opt=paths=import")
$protocArgs += @("--goone_out=$OutDir", "--goone_opt=module=$Module", "--goone_opt=paths=import")
$protocArgs += $protos

& $protoc @protocArgs
Write-Host "[proto_goone] done"


