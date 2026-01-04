Param(
  [string]$Prefix = "CMD_",
  [string]$Exact = ""
)

$ErrorActionPreference = "Stop"

Push-Location (Split-Path $PSScriptRoot -Parent)
try {
  if ($Exact -ne "") {
    go run .\tools\cmd\dumpg1cmd -exact $Exact
  }
  else {
    go run .\tools\cmd\dumpg1cmd -prefix $Prefix
  }
}
finally {
  Pop-Location
}


