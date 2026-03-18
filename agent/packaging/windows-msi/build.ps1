param(
  [string]$WixBin = "C:\Program Files (x86)\WiX Toolset v3.14\bin",
  [string]$OutMsi = ".\nocguardian-agent-windows-x64.msi"
)

$ErrorActionPreference = "Stop"

$candle = Join-Path $WixBin "candle.exe"
$light  = Join-Path $WixBin "light.exe"

if (!(Test-Path $candle)) { throw "candle.exe not found: $candle" }
if (!(Test-Path $light))  { throw "light.exe not found:  $light" }

# GUIDs: substitua no installer.wxs antes de buildar (UpgradeCode e Component Guid)
Write-Host "Building MSI..."
& $candle ".\installer.wxs" -ext WixUtilExtension
& $light  ".\installer.wixobj" -ext WixUtilExtension -o $OutMsi

Write-Host "OK => $OutMsi"
