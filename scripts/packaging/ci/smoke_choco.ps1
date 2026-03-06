#!/usr/bin/env pwsh

param(
  [string]$ManifestPath = "dist/release-manifest.json",
  [string]$DistDir = "dist",
  [string]$OutputDir = "",
  [string]$PackageName = "surveiller"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-RepoRoot {
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

  if ([string]::IsNullOrWhiteSpace($PathValue)) {
    return $PathValue
  }
  if ([System.IO.Path]::IsPathRooted($PathValue)) {
    return $PathValue
  }
  return [System.IO.Path]::GetFullPath((Join-Path $Root $PathValue))
}

function Require-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "required command is not installed: $Name"
  }
}

$repoRoot = Resolve-RepoRoot
$manifestAbs = Resolve-RepoPath -Root $repoRoot -PathValue $ManifestPath
$distAbs = Resolve-RepoPath -Root $repoRoot -PathValue $DistDir

if ([string]::IsNullOrWhiteSpace($OutputDir)) {
  $OutputDir = Join-Path $DistDir "packages/choco-smoke"
}
$outputAbs = Resolve-RepoPath -Root $repoRoot -PathValue $OutputDir
$checksumsAbs = Join-Path $distAbs "checksums.txt"
$localManifestAbs = Join-Path $distAbs "release-manifest.choco-smoke.json"
$buildScript = Join-Path $repoRoot "scripts/packaging/build_choco_package.ps1"

if (-not (Test-Path $manifestAbs)) {
  throw "required file not found: $manifestAbs"
}
if (-not (Test-Path $checksumsAbs)) {
  throw "required file not found: $checksumsAbs"
}
if (-not (Test-Path $buildScript)) {
  throw "required file not found: $buildScript"
}

Require-Command -Name "choco"

$manifest = Get-Content -Path $manifestAbs -Raw | ConvertFrom-Json
$version = [string]$manifest.version
if ([string]::IsNullOrWhiteSpace($version)) {
  throw "manifest missing version"
}

$windowsBinary = Join-Path $distAbs "surveiller-windows-amd64.exe"
if (-not (Test-Path $windowsBinary)) {
  throw "required file not found: $windowsBinary"
}

$chocoVersion = [string]$manifest.package_versions.choco
if ([string]::IsNullOrWhiteSpace($chocoVersion)) {
  throw "manifest missing package_versions.choco"
}

$manifestCopy = $manifest | ConvertTo-Json -Depth 50 | ConvertFrom-Json
$windowsArtifact = $manifestCopy.artifacts | Where-Object { $_.name -eq "surveiller-windows-amd64.exe" } | Select-Object -First 1
if ($null -eq $windowsArtifact) {
  throw "manifest missing windows amd64 artifact"
}

$windowsBinaryUri = [System.Uri]::new($windowsBinary).AbsoluteUri
$windowsArtifact.url = $windowsBinaryUri
$manifestCopy.release_base_url = ([System.Uri]::new($distAbs + [System.IO.Path]::DirectorySeparatorChar)).AbsoluteUri.TrimEnd('/')
$manifestCopy.release_date = [DateTime]::UtcNow.ToString("o")

$manifestCopy | ConvertTo-Json -Depth 50 | Set-Content -Path $localManifestAbs -Encoding utf8

New-Item -ItemType Directory -Path $outputAbs -Force | Out-Null

& $buildScript `
  -ManifestPath $localManifestAbs `
  -DistDir $distAbs `
  -OutputDir $outputAbs `
  -ChecksumsFile $checksumsAbs

$nupkg = Get-ChildItem -Path $outputAbs -Filter "surveiller.$chocoVersion*.nupkg" -ErrorAction SilentlyContinue |
  Sort-Object LastWriteTime -Descending |
  Select-Object -First 1
if ($null -eq $nupkg) {
  throw "generated nupkg not found in $outputAbs"
}

& choco uninstall $PackageName -y --no-progress | Out-Null

& choco install $PackageName `
  -y `
  --pre `
  --force `
  --source $outputAbs `
  --no-progress

if ($LASTEXITCODE -ne 0) {
  throw "choco install failed with exit code $LASTEXITCODE"
}

& $PackageName -version | Out-Null

Write-Host "[packaging] choco smoke install passed for package $PackageName"
