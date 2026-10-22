# One-line install (PowerShell):
#   iex (irm https://raw.githubusercontent.com/amurru/filetools/main/install.ps1)
$ErrorActionPreference = "Stop"

$repo = "amurru/filetools"
$asset = "filetools-windows-amd64.exe"
$installDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\filetools" }

$api = "https://api.github.com/repos/$repo/releases/latest"
$release = Invoke-RestMethod -Uri $api -Headers @{ "User-Agent" = "filetools-installer" }
$assetInfo = $release.assets | Where-Object { $_.name -eq $asset } | Select-Object -First 1
if (-not $assetInfo) {
    throw "Could not find a release asset named $asset"
}

New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$target = Join-Path $installDir "filetools.exe"
Write-Host "Downloading $($assetInfo.browser_download_url)..."
Invoke-WebRequest -Uri $assetInfo.browser_download_url -OutFile $target

# Add to user PATH if not already present
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your PATH (restart your shell to use it)."
}

Write-Host "Installed filetools to $target"
& $target version