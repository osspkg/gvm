$ErrorActionPreference = "Stop"

$repository = "osspkg/gvm"
$gvmHome = if ($env:GVM_HOME) { $env:GVM_HOME } else { Join-Path $HOME ".gvm" }
$binDir = Join-Path $gvmHome "bin"
$cacheDir = Join-Path $gvmHome ".cache"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("gvm-" + [System.Guid]::NewGuid().ToString("N"))

try {
    New-Item -ItemType Directory -Force -Path $tempDir, $binDir, (Join-Path $cacheDir "bin"), (Join-Path $cacheDir "pkg"), (Join-Path $cacheDir "src") | Out-Null
    $release = Invoke-RestMethod -Headers @{ "Accept" = "application/vnd.github+json"; "User-Agent" = "gvm-installer" } -Uri "https://api.github.com/repos/$repository/releases/latest"
    $version = $release.tag_name.TrimStart("v")
    $architecture = switch ([System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture) {
        "X64" { "amd64" }
        "Arm64" { "arm64" }
        default { throw "unsupported architecture" }
    }
    $archiveName = "gvm_${version}_windows_${architecture}.zip"
    $archivePath = Join-Path $tempDir $archiveName
    Invoke-WebRequest -Headers @{ "User-Agent" = "gvm-installer" } -Uri "https://github.com/$repository/releases/download/$($release.tag_name)/$archiveName" -OutFile $archivePath
    Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force
    Copy-Item (Join-Path $tempDir "gvm.exe") (Join-Path $binDir "gvm.exe") -Force
    Copy-Item (Join-Path $tempDir "go.exe") (Join-Path $binDir "go.exe") -Force
    Copy-Item (Join-Path $tempDir "gofmt.exe") (Join-Path $binDir "gofmt.exe") -Force

    [Environment]::SetEnvironmentVariable("GVM_HOME", $gvmHome, "User")
    $pathEntry = $binDir.TrimEnd("\")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $userParts = @($userPath -split ";" | Where-Object { $_ -and $_ -ne $pathEntry })
    [Environment]::SetEnvironmentVariable("Path", ((@($pathEntry) + $userParts) -join ";"), "User")

    $env:GVM_HOME = $gvmHome
    $processParts = @($env:Path -split ";" | Where-Object { $_ -and $_ -ne $pathEntry })
    $env:Path = ((@($pathEntry) + $processParts) -join ";")

    if (-not (Test-Path $PROFILE)) {
        New-Item -ItemType File -Force -Path $PROFILE | Out-Null
    }
    $profileLines = @(Get-Content -LiteralPath $PROFILE)
    $profileOutput = New-Object System.Collections.Generic.List[string]
    $insideGvmBlock = $false
    foreach ($line in $profileLines) {
        if ($line -eq '# >>> gvm >>>') {
            $insideGvmBlock = $true
            continue
        }
        if ($insideGvmBlock -and $line -eq '# <<< gvm <<<') {
            $insideGvmBlock = $false
            continue
        }
        if (-not $insideGvmBlock) {
            [void]$profileOutput.Add($line)
        }
    }
    if ($insideGvmBlock) {
        throw "profile contains an unterminated gvm block: $PROFILE"
    }
    $profileBlock = @(
        "",
        '# >>> gvm >>>',
        '$env:GVM_HOME = if ($env:GVM_HOME) { $env:GVM_HOME } else { Join-Path $HOME ''.gvm'' }',
        '$gvmBin = Join-Path $env:GVM_HOME ''bin''',
        '$gvmCacheBin = Join-Path $env:GVM_HOME ''.cache\bin''',
        '$gvmPathParts = @($env:Path -split '';'' | Where-Object { $_ -and $_ -ne $gvmCacheBin -and $_ -ne $gvmBin })',
        '$env:Path = ((@($gvmCacheBin, $gvmBin) + $gvmPathParts) -join '';'' )',
        '# <<< gvm <<<'
    )
    Set-Content -LiteralPath $PROFILE -Value (@($profileOutput.ToArray()) + $profileBlock)
    Write-Output "gvm $($release.tag_name) installed in $gvmHome"
}
finally {
    if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
}
