# QuecTool deploy script (PowerShell). Workstation-side companion to deploy.sh
# for Windows operators who don't have WSL / Git Bash. Pushes a release tarball
# to an adb-attached Quectel modem and runs the on-device installer.
#
#   .\deploy.ps1                                       # latest GitHub release
#   .\deploy.ps1 v2.0.0                                # specific tag
#   .\deploy.ps1 .\quectool-v2.0.0-armv7.tar.gz        # local tarball
#
# Requirements on the workstation: adb.exe in PATH, PowerShell 5+ (Windows 10
# ships PS 5.1; 7+ also works).
# Requirements on the modem: adb shell as root, /usrdata writable.

[CmdletBinding()]
param(
    [Parameter(Position=0)]
    [string]$Source = "latest"
)

$ErrorActionPreference = "Stop"

function Die([string]$msg) { Write-Error $msg; exit 1 }
function Need([string]$cmd) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Die "missing required tool: $cmd"
    }
}

Need adb

$Repo = if ($env:QUECTOOL_REPO) { $env:QUECTOOL_REPO } else { "snowzach/quectool" }
$Work = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ("quectool-deploy-" + [Guid]::NewGuid().ToString("N")))

try {
    # --- Resolve tarball -----------------------------------------------------
    if (Test-Path -LiteralPath $Source -PathType Leaf) {
        $Tarball = (Resolve-Path -LiteralPath $Source).Path
        Write-Host "==> using local tarball: $Tarball"
    }
    else {
        $apiUrl = if ($Source -eq "latest") {
            "https://api.github.com/repos/$Repo/releases/latest"
        } else {
            "https://api.github.com/repos/$Repo/releases/tags/$Source"
        }
        Write-Host "==> resolving release: $Source"

        # GitHub's API requires a User-Agent. Invoke-RestMethod sets a sane
        # default but be explicit.
        $headers = @{ "User-Agent" = "quectool-deploy-ps1" }
        try {
            $rel = Invoke-RestMethod -Uri $apiUrl -Headers $headers
        } catch {
            Die "failed to fetch release metadata from $apiUrl: $_"
        }

        $tarAsset = $rel.assets | Where-Object { $_.name -like "*armv7.tar.gz" } | Select-Object -First 1
        $shaAsset = $rel.assets | Where-Object { $_.name -like "*armv7.tar.gz.sha256" } | Select-Object -First 1
        if (-not $tarAsset) { Die "no armv7.tar.gz asset on release $Source" }

        $Tarball = Join-Path $Work $tarAsset.name
        Write-Host "==> downloading $($tarAsset.browser_download_url)"
        Invoke-WebRequest -Uri $tarAsset.browser_download_url -OutFile $Tarball -UseBasicParsing

        if ($shaAsset) {
            Write-Host "==> verifying sha256"
            $shaPath = Join-Path $Work $shaAsset.name
            Invoke-WebRequest -Uri $shaAsset.browser_download_url -OutFile $shaPath -UseBasicParsing
            # The .sha256 file format is "<hex>  <filename>" (gnu sha256sum).
            $expected = ((Get-Content -LiteralPath $shaPath -Raw) -split '\s+')[0].ToLower()
            $actual = (Get-FileHash -LiteralPath $Tarball -Algorithm SHA256).Hash.ToLower()
            if ($expected -ne $actual) {
                Die "sha256 mismatch on $Tarball (expected $expected, got $actual)"
            }
        } else {
            Write-Host "==> no .sha256 published; skipping checksum verification"
        }
    }

    # --- Push & run on modem -------------------------------------------------
    Write-Host "==> waiting for adb device"
    & adb wait-for-device
    if ($LASTEXITCODE -ne 0) { Die "adb wait-for-device failed" }

    $remoteTar = "/tmp/quectool-deploy.tar.gz"
    $remoteDir = "/tmp/quectool-deploy"

    Write-Host "==> adb push -> $remoteTar"
    & adb push $Tarball $remoteTar | Out-Null
    if ($LASTEXITCODE -ne 0) { Die "adb push failed" }

    Write-Host "==> running install.sh on modem"
    $cmd = "set -e; rm -rf $remoteDir; mkdir -p $remoteDir; cd $remoteDir && tar xzf $remoteTar && sh ./install.sh; cd / && rm -rf $remoteDir $remoteTar"
    & adb shell $cmd
    if ($LASTEXITCODE -ne 0) { Die "install.sh failed on modem (see output above)" }

    Write-Host ""
    Write-Host "Deploy complete."
}
finally {
    if (Test-Path -LiteralPath $Work) {
        Remove-Item -LiteralPath $Work -Recurse -Force -ErrorAction SilentlyContinue
    }
}
