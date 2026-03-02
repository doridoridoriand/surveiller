#!/usr/bin/env pwsh

param(
  [string]$Version = "",
  [string]$ManifestPath = "dist/release-manifest.json",
  [string]$DistDir = "dist",
  [string]$PackagePath = "",
  [string]$StatusFile = "",
  [string]$DryRun = "true",
  [int]$Attempt = 1,
  [string]$SourceUrl = ""
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

function Is-True {
  param([string]$Value)
  $normalized = ($Value ?? "").Trim().ToLowerInvariant()
  return @("1", "true", "yes", "on") -contains $normalized
}

function Write-Status {
  param(
    [string]$Path,
    [string]$Manager,
    [string]$ReleaseVersion,
    [string]$State,
    [string]$Target,
    [int]$AttemptNumber,
    [int]$ArtifactCount,
    [Nullable[DateTime]]$PublishedAt,
    [string]$ErrorMessage
  )

  $statusObject = [ordered]@{
    manager = $Manager
    version = $ReleaseVersion
    state = $State
    repository_target = $Target
    attempt = $AttemptNumber
    artifact_count = $ArtifactCount
    published_at = if ($null -eq $PublishedAt) { $null } else { $PublishedAt.Value.ToUniversalTime().ToString("o") }
    error_message = if ([string]::IsNullOrWhiteSpace($ErrorMessage)) { $null } else { $ErrorMessage }
  }

  $dir = Split-Path -Parent $Path
  if (-not [string]::IsNullOrWhiteSpace($dir)) {
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
  }

  $statusObject | ConvertTo-Json -Depth 5 | Set-Content -Path $Path -Encoding utf8
}

$repoRoot = Resolve-RepoRoot
$manifestAbs = Resolve-RepoPath -Root $repoRoot -PathValue $ManifestPath
$distAbs = Resolve-RepoPath -Root $repoRoot -PathValue $DistDir

if ([string]::IsNullOrWhiteSpace($StatusFile)) {
  $StatusFile = Join-Path $DistDir "publication-status/choco.json"
}
$statusAbs = Resolve-RepoPath -Root $repoRoot -PathValue $StatusFile

if (-not (Test-Path $manifestAbs)) {
  throw "required file not found: $manifestAbs"
}

$manifest = Get-Content -Path $manifestAbs -Raw | ConvertFrom-Json
$releaseVersion = if ([string]::IsNullOrWhiteSpace($Version)) { [string]$manifest.version } else { $Version }
if ([string]::IsNullOrWhiteSpace($releaseVersion)) {
  throw "version is required"
}

$chocoVersion = [string]$manifest.package_versions.choco
if ([string]::IsNullOrWhiteSpace($chocoVersion)) {
  throw "manifest missing package_versions.choco"
}

if ([string]::IsNullOrWhiteSpace($PackagePath)) {
  $candidate = Join-Path $distAbs "packages/choco/surveiller.$chocoVersion.nupkg"
  if (Test-Path $candidate) {
    $PackagePath = $candidate
  }
  else {
    $found = Get-ChildItem -Path (Join-Path $distAbs "packages/choco") -Filter "surveiller.$chocoVersion*.nupkg" -ErrorAction SilentlyContinue |
      Sort-Object LastWriteTime -Descending |
      Select-Object -First 1
    if ($null -ne $found) {
      $PackagePath = $found.FullName
    }
  }
}

$packageAbs = Resolve-RepoPath -Root $repoRoot -PathValue $PackagePath
if ([string]::IsNullOrWhiteSpace($packageAbs) -or -not (Test-Path $packageAbs)) {
  Write-Status -Path $statusAbs -Manager "choco" -ReleaseVersion $releaseVersion -State "failed" `
    -Target ($SourceUrl ?? "") -AttemptNumber $Attempt -ArtifactCount 0 -PublishedAt $null `
    -ErrorMessage "nupkg not found for choco version $chocoVersion"
  throw "nupkg not found for choco version $chocoVersion"
}

$state = "failed"
$errorMessage = ""
$publishedAt = $null
$target = if ([string]::IsNullOrWhiteSpace($SourceUrl)) {
  if ([string]::IsNullOrWhiteSpace($env:CHOCO_SOURCE_URL)) { "https://push.chocolatey.org/" } else { $env:CHOCO_SOURCE_URL }
}
else { $SourceUrl }

if (Is-True $DryRun) {
  $state = "queued"
}
else {
  if (-not (Get-Command choco -ErrorAction SilentlyContinue)) {
    $state = "failed"
    $errorMessage = "choco command is not installed"
  }
  elseif ([string]::IsNullOrWhiteSpace($env:CHOCO_API_KEY)) {
    $state = "failed"
    $errorMessage = "CHOCO_API_KEY is not set"
  }
  else {
    try {
      & choco push $packageAbs --source $target --api-key $env:CHOCO_API_KEY --timeout 900 | Out-Null
      if ($LASTEXITCODE -ne 0) {
        throw "choco push exited with code $LASTEXITCODE"
      }
      $state = "published"
      $publishedAt = [DateTime]::UtcNow
    }
    catch {
      $state = "failed"
      $errorMessage = $_.Exception.Message
    }
  }
}

Write-Status -Path $statusAbs -Manager "choco" -ReleaseVersion $releaseVersion -State $state `
  -Target $target -AttemptNumber $Attempt -ArtifactCount 1 -PublishedAt $publishedAt `
  -ErrorMessage $errorMessage

if ($state -eq "failed") {
  throw "choco publish failed: $errorMessage"
}

Write-Host "[packaging] choco publication state=$state status_file=$statusAbs"
