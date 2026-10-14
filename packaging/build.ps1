param(
    [string]$Version = "0.1.0-dev",
    [switch]$Installer
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
$stage = Join-Path $dist "platty"

Push-Location $root

try {
    Remove-Item $dist -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force $stage | Out-Null

    $numeric = ($Version -split "-")[0] + ".0"

    Push-Location "cmd/platty"
    go run github.com/tc-hib/go-winres@v0.3.3 make --arch amd64 --product-version $numeric --file-version $numeric
    Pop-Location

    $ldflags = "-H windowsgui -s -w -X github.com/plattyssh/platty/internal/version.Version=$Version"
    go build -trimpath -ldflags $ldflags -o (Join-Path $stage "platty.exe") ./cmd/platty

    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }

    Copy-Item LICENSE, README.md, THIRD_PARTY_NOTICES.md $stage
    Compress-Archive -Path (Join-Path $stage "*") -DestinationPath (Join-Path $dist "platty-$Version-windows-amd64.zip")

    if ($Installer) {
        $iscc = Get-Command iscc -ErrorAction SilentlyContinue

        if (-not $iscc) {
            choco install innosetup -y --no-progress | Out-Null
            $iscc = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe"
        }

        & $iscc "/DVersion=$Version" "/DSourceDir=$stage" "/O$dist" "packaging/platty.iss"

        if ($LASTEXITCODE -ne 0) {
            throw "Inno Setup failed"
        }
    }

    Get-ChildItem $dist -File | Select-Object Name, Length
}
finally {
    Pop-Location
}
