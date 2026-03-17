param(
  [string]$WixBin = "C:\Program Files (x86)\WiX Toolset v3.11\bin",
  [string]$OutMsi = ".\nocguardian-agent-windows-x64.msi"
)

$ErrorActionPreference = "Stop"

$candle = Join-Path $WixBin "candle.exe"
$light  = Join-Path $WixBin "light.exe"

if (!(Test-Path $candle)) { throw "candle.exe not found: $candle" }
if (!(Test-Path $light))  { throw "light.exe not found:  $light" }

# Ajuste WixCAPath para apontar para WixCA.dll
# Em WiX v3.11, normalmente está em:
#   C:\Program Files (x86)\WiX Toolset v3.11\bin\WixCA.dll
$WixCAPath = Join-Path $WixBin "WixCA.dll"
if (!(Test-Path $WixCAPath)) { throw "WixCA.dll not found: $WixCAPath" }

# GUIDs: substitua no installer.wxs antes de buildar (UpgradeCode e Component Guid)
Write-Host "Building MSI..."
& $candle ".\installer.wxs" -ext WixUtilExtension -dWixCAPath="$WixCAPath"
& $light  ".\installer.wixobj" -ext WixUtilExtension -o $OutMsi

Write-Host "OK => $OutMsi"
