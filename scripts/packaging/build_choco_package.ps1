#!/usr/bin/env pwsh

param(
  [string]$ManifestPath = "dist/release-manifest.json",
  [string]$TemplatePath = "packaging/choco/surveiller.nuspec.tmpl",
  [string]$DistDir = "dist",
  [string]$OutputDir = "",
  [string]$ChecksumsFile = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-RepoRoot {
  $repoRoot = & git rev-parse --show-toplevel 2>$null
  if ($LASTEXITCODE -eq 0 -and $repoRoot) {
    return $repoRoot.Trim()
  }
  return (Get-Location).Path
}

function Resolve-RepoPath {
  param(
    [string]$Root,
    [string]$PathValue
  )

  if ([System.IO.Path]::IsPathRooted($PathValue)) {
    return $PathValue
  }
  return [System.IO.Path]::GetFullPath((Join-Path $Root $PathValue))
}

function Get-RelativeToDist {
  param(
    [string]$DistRoot,
    [string]$FilePath
  )

  $relative = [System.IO.Path]::GetRelativePath($DistRoot, $FilePath)
  return $relative.Replace('\', '/')
}

function Upsert-Checksum {
  param(
    [string]$ChecksumsPath,
    [string]$ArtifactName,
    [string]$Checksum
  )

  $pattern = '^[a-fA-F0-9]{64}$'
  if ($Checksum -notmatch $pattern) {
    throw "invalid SHA256 for ${ArtifactName}: $Checksum"
  }

  $rows = @()
  if (Test-Path $ChecksumsPath) {
    $rows = Get-Content -Path $ChecksumsPath
  }

  $filtered = foreach ($line in $rows) {
    if ($line -match '^[a-fA-F0-9]{64}\s+\*?(.+)$') {
      $name = $Matches[1]
      if ($name -eq $ArtifactName) {
        continue
      }
    }
    $line
  }

  $filtered += "$Checksum  $ArtifactName"

  $sorted = @()
  $pairs = foreach ($line in $filtered) {
    if ($line -match '^([a-fA-F0-9]{64})\s+\*?(.+)$') {
      [PSCustomObject]@{
        Name = $Matches[2]
        SHA  = $Matches[1].ToLowerInvariant()
      }
    }
  }

  $sorted = $pairs | Sort-Object -Property Name | ForEach-Object {
    "$($_.SHA)  $($_.Name)"
  }

  Set-Content -Path $ChecksumsPath -Value $sorted
}

function Write-PackagingLog {
  param([string]$Message)
  Write-Host "[packaging] $Message"
}

$repoRoot = Get-RepoRoot
$manifestAbs = Resolve-RepoPath -Root $repoRoot -PathValue $ManifestPath
$templateAbs = Resolve-RepoPath -Root $repoRoot -PathValue $TemplatePath
$distDirAbs = Resolve-RepoPath -Root $repoRoot -PathValue $DistDir

if ([string]::IsNullOrWhiteSpace($OutputDir)) {
  $OutputDir = Join-Path $DistDir "packages/choco"
}
if ([string]::IsNullOrWhiteSpace($ChecksumsFile)) {
  $ChecksumsFile = Join-Path $DistDir "checksums.txt"
}

$outputDirAbs = Resolve-RepoPath -Root $repoRoot -PathValue $OutputDir
$checksumsAbs = Resolve-RepoPath -Root $repoRoot -PathValue $ChecksumsFile

if (-not (Test-Path $manifestAbs)) {
  throw "required file not found: $manifestAbs"
}
if (-not (Test-Path $templateAbs)) {
  throw "required file not found: $templateAbs"
}

$manifest = Get-Content -Path $manifestAbs -Raw | ConvertFrom-Json

$releaseVersion = [string]$manifest.version
if ([string]::IsNullOrWhiteSpace($releaseVersion)) {
  throw "manifest missing version"
}
$versionNoV = $releaseVersion.TrimStart('v')
$chocoVersion = [string]$manifest.package_versions.choco
if ([string]::IsNullOrWhiteSpace($chocoVersion)) {
  throw "manifest missing package_versions.choco"
}

$winArtifact = $manifest.artifacts | Where-Object { $_.name -eq "surveiller-windows-amd64.exe" } | Select-Object -First 1
if ($null -eq $winArtifact) {
  throw "manifest missing windows amd64 artifact"
}

$winURL = [string]$winArtifact.url
$winSHA = [string]$winArtifact.sha256
if ($winSHA -notmatch '^[a-f0-9]{64}$') {
  throw "invalid windows artifact SHA256 in manifest: $winSHA"
}

New-Item -ItemType Directory -Path $outputDirAbs -Force | Out-Null
if (-not (Test-Path $checksumsAbs)) {
  New-Item -ItemType File -Path $checksumsAbs -Force | Out-Null
}

$tmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("surveiller-choco-" + [Guid]::NewGuid().ToString("N"))
$tmpTools = Join-Path $tmpRoot "tools"

New-Item -ItemType Directory -Path $tmpTools -Force | Out-Null

try {
  $templateContent = Get-Content -Path $templateAbs -Raw
  $nuspecContent = $templateContent.Replace("{{ .ChocolateyVersion }}", $chocoVersion).Replace("{{ .Version }}", $versionNoV)

  if ($nuspecContent -match '\{\{\s*\.') {
    throw "unresolved placeholders remain in nuspec template"
  }

  $nuspecPath = Join-Path $tmpRoot "surveiller.nuspec"
  Set-Content -Path $nuspecPath -Value $nuspecContent -Encoding utf8

  $installScript = @"
`$ErrorActionPreference = 'Stop'
`$packageName = 'surveiller'
`$toolsDir = Split-Path -Parent `$MyInvocation.MyCommand.Definition
`$fileFullPath = Join-Path `$toolsDir 'surveiller.exe'

`$packageArgs = @{
  packageName    = `$packageName
  fileFullPath   = `$fileFullPath
  url64bit       = '$winURL'
  checksum64     = '$winSHA'
  checksumType64 = 'sha256'
}

Get-ChocolateyWebFile @packageArgs
"@
  Set-Content -Path (Join-Path $tmpTools "chocolateyinstall.ps1") -Value $installScript -Encoding utf8

  $verificationText = @"
VERIFICATION
1. Download URL: $winURL
2. SHA256: $winSHA
3. Install command: choco install surveiller
"@
  Set-Content -Path (Join-Path $tmpTools "VERIFICATION.txt") -Value $verificationText -Encoding utf8

  $licensePath = Join-Path $repoRoot "LICENSE"
  if (Test-Path $licensePath) {
    Copy-Item -Path $licensePath -Destination (Join-Path $tmpTools "LICENSE.txt") -Force
  }

  $packagePath = Join-Path $outputDirAbs "surveiller.$chocoVersion.nupkg"
  if (Get-Command choco -ErrorAction SilentlyContinue) {
    Push-Location $tmpRoot
    try {
      & choco pack "surveiller.nuspec" --outputdirectory $outputDirAbs | Out-Null
      if ($LASTEXITCODE -ne 0) {
        throw "choco pack failed"
      }
      $packed = Get-ChildItem -Path $outputDirAbs -Filter "surveiller.$chocoVersion*.nupkg" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
      if ($null -eq $packed) {
        throw "choco pack did not produce nupkg"
      }
      $packagePath = $packed.FullName
    }
    finally {
      Pop-Location
    }
  }
  else {
    $zipPath = [System.IO.Path]::ChangeExtension($packagePath, ".zip")
    if (Test-Path $zipPath) {
      Remove-Item -Path $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $tmpRoot "*") -DestinationPath $zipPath -Force
    if (Test-Path $packagePath) {
      Remove-Item -Path $packagePath -Force
    }
    Move-Item -Path $zipPath -Destination $packagePath
  }

  Copy-Item -Path $nuspecPath -Destination (Join-Path $outputDirAbs "surveiller.nuspec") -Force
  Copy-Item -Path (Join-Path $tmpTools "chocolateyinstall.ps1") -Destination (Join-Path $outputDirAbs "chocolateyinstall.ps1") -Force

  $hash = (Get-FileHash -Algorithm SHA256 -Path $packagePath).Hash.ToLowerInvariant()
  $artifactName = Get-RelativeToDist -DistRoot $distDirAbs -FilePath $packagePath
  Upsert-Checksum -ChecksumsPath $checksumsAbs -ArtifactName $artifactName -Checksum $hash

  Write-PackagingLog "Chocolatey payload generated at $packagePath"
}
finally {
  if (Test-Path $tmpRoot) {
    Remove-Item -Path $tmpRoot -Recurse -Force
  }
}
