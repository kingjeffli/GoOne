Param(
  [string]$Module = "github.com/Iori372552686/GoOne",
  [string]$OutDir = ".",
  [string]$ProtoRoot = "api/proto",
  [string]$BinDir = ".bin"
)

$ErrorActionPreference = "Stop"

Write-Host "[proto_goone] build plugins -> $BinDir"
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

go build -o (Join-Path $BinDir "protoc-gen-goone.exe") .\tools\protoc-gen-goone
go build -o (Join-Path $BinDir "protoc-gen-go.exe") "google.golang.org/protobuf/cmd/protoc-gen-go"

try {
  $protoc = (Get-Command protoc -ErrorAction Stop).Source
} catch {
  Write-Host "[proto_goone] ERROR: protoc not found on PATH. Install protoc (Windows) or run ./scripts/proto_goone.sh in WSL." -ForegroundColor Red
  throw
}

$include = @($ProtoRoot)

# Add vendored include dirs if present (mainly for CI / portable setups)
$cand = @(
  "lib/contrib/protoc/protoc-30.1-win64/include",
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

$args = @()
foreach ($i in $include) { $args += @("-I", $i) }

$args += @("--plugin=protoc-gen-go=$(Resolve-Path (Join-Path $BinDir "protoc-gen-go.exe"))")
$args += @("--plugin=protoc-gen-goone=$(Resolve-Path (Join-Path $BinDir "protoc-gen-goone.exe"))")
$args += @("--go_out=$OutDir", "--go_opt=module=$Module", "--go_opt=paths=import")
$args += @("--goone_out=$OutDir", "--goone_opt=module=$Module", "--goone_opt=paths=import")
$args += $protos

& $protoc @args
Write-Host "[proto_goone] done"


