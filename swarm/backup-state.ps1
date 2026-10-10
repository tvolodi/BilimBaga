<#
 Swarm state backup. Safe by construction: it only ever COPIES, and the only deletion is of its own dated folders.
 1. Copies swarm/state/*.json, swarm/roster.json and swarm/roles/*.settings.json to
    <BackupRoot>\yyyyMMdd-HHmmss (default %USERPROFILE%\.swarm-backups\bilimbaga, outside the repo).
 2. Prunes: only direct children of BackupRoot whose name matches ^\d{8}-\d{6}$ (real directories, not links),
    keeping the newest -Keep (20). Anything else in BackupRoot is never touched. Pruning runs only after a
    successful copy.
 3. Publishing is OFF by default (#284: the repo is public, so nothing swarm-related is pushed anywhere).
    Only an explicit -PushRemote (tests: a local bare repo) reaches the snapshot code, which pushes a text
    snapshot (swarm-snapshot.txt: roster + open issue counts and titles, tokens scrubbed) to branch swarm-state of
    THAT remote, through a throwaway temp repo, so the working checkout is never switched. A failed push only warns.
 -MinIntervalMin N: do nothing if the newest dated folder is younger than N minutes (used by ensure-up).
 Usage: powershell -File swarm\backup-state.ps1 [-WhatIf] [-BackupRoot dir] [-Keep 20] [-PushRemote url (tests only)]
        [-QueueJsonFile stub.json] [-MinIntervalMin 60] [-NowUtc <datetime>]
#>
[CmdletBinding(SupportsShouldProcess)]
param(
  [string]$BackupRoot = (Join-Path $env:USERPROFILE '.swarm-backups\bilimbaga'),
  [int]$Keep = 20,
  [switch]$NoPush,                  # accepted for compatibility; publishing is already off by default (#284)
  [string]$PushRemote,              # tests only: a local bare repo. Never defaults to origin (#284).
  [string]$QueueJsonFile,           # test stub replacing `gh issue list`
  [int]$MinIntervalMin = 0,
  [datetime]$NowUtc = ([datetime]::UtcNow)
)
$ErrorActionPreference = 'Stop'
$env:GIT_TERMINAL_PROMPT = '0'   # never wait for a credential prompt
$env:GCM_INTERACTIVE = 'never'
Remove-Item Env:GIT_ASKPASS, Env:SSH_ASKPASS -WhatIf:$false -ErrorAction SilentlyContinue
$gitSafe = @('-c', 'core.hooksPath=NUL', '-c', 'commit.gpgsign=false', '-c', 'tag.gpgsign=false', '-c', 'core.autocrlf=false', '-c', 'credential.interactive=never')
. (Join-Path $PSScriptRoot 'lib.ps1')
$roster   = Get-Roster
$stateDir = Get-StateDir $roster
$NowUtc   = $NowUtc.ToUniversalTime()
$namePat  = '^\d{8}-\d{6}$'
if ($Keep -lt 1) { Write-Error 'Keep must be >= 1'; exit 2 }
$rootFull = [IO.Path]::GetFullPath($BackupRoot).TrimEnd('\')

function Get-DatedDirs {
  if (-not (Test-Path -LiteralPath $BackupRoot -PathType Container)) { return @() }
  @(Get-ChildItem -LiteralPath $BackupRoot -Directory -Force | Where-Object { $_.Name -match $namePat -and -not ($_.Attributes -band [IO.FileAttributes]::ReparsePoint) } | Sort-Object Name)
}

if ($MinIntervalMin -gt 0) {
  $last = Get-DatedDirs | Select-Object -Last 1
  if ($last -and ($NowUtc - $last.CreationTimeUtc).TotalMinutes -lt $MinIntervalMin) { Write-Host "backup: newest folder $($last.Name) is younger than $MinIntervalMin min, skipping"; exit 0 }
}

# 1. copy
$dest = Join-Path $BackupRoot ($NowUtc.ToString('yyyyMMdd-HHmmss'))
$copied = $false
$sources = @()
if (Test-Path -LiteralPath $stateDir) { $sources += @{ From = @(Get-ChildItem -LiteralPath $stateDir -Filter '*.json' -File); To = 'state' } }
$sources += @{ From = @(Get-Item -LiteralPath (Join-Path $PSScriptRoot 'roster.json')); To = '.' }
$rolesDir = Join-Path $PSScriptRoot 'roles'
if (Test-Path -LiteralPath $rolesDir) { $sources += @{ From = @(Get-ChildItem -LiteralPath $rolesDir -Filter '*.settings.json' -File); To = 'roles' } }
Write-Host "backup: $dest"
foreach ($s in $sources) { foreach ($f in $s.From) { Write-Host "  copy $($f.Name) -> $($s.To)" } }
if (Test-Path -LiteralPath $dest) { Write-Warning "backup: $dest already exists, not overwriting"; exit 0 }
if ($PSCmdlet.ShouldProcess($dest, 'copy state, roster and role settings')) {
  try {
    New-Item -ItemType Directory -Path $dest -Force | Out-Null
    foreach ($s in $sources) {
      $td = if ($s.To -eq '.') { $dest } else { Join-Path $dest $s.To }
      if (-not (Test-Path -LiteralPath $td)) { New-Item -ItemType Directory -Path $td -Force | Out-Null }
      foreach ($f in $s.From) { Copy-Item -LiteralPath $f.FullName -Destination $td -Force }
    }
    $copied = $true
  } catch { Remove-Item -LiteralPath $dest -Recurse -Force -ErrorAction SilentlyContinue; Write-Error "backup copy failed: $_"; exit 1 }  # $dest is our own new dated folder
}

# 2. prune (own dated folders only, only after a successful copy)
$dated = @(Get-DatedDirs)
$excess = if ($dated.Count -gt $Keep) { @($dated | Select-Object -First ($dated.Count - $Keep)) } else { @() }
foreach ($d in $excess) {
  if ($d.Name -notmatch $namePat -or $d.Parent.FullName.TrimEnd('\') -ne $rootFull) { continue }  # belt and braces
  Write-Host "  prune $($d.Name)"
  if ($copied -and $PSCmdlet.ShouldProcess($d.FullName, 'remove old dated backup')) { Remove-Item -LiteralPath $d.FullName -Recurse -Force }
}

# 3. snapshot to branch swarm-state of an explicit remote only (#284). The default is no publishing at all.
if ($NoPush -or -not $PushRemote) {
  Write-Host 'snapshot publishing is off (#284): local backup only; -PushRemote is for tests'
  exit 0
}
function New-Snapshot {
  $b = New-Object System.Text.StringBuilder
  [void]$b.AppendLine("BilimBaga swarm snapshot $($NowUtc.ToString('yyyy-MM-ddTHH:mm:ssZ'))")
  [void]$b.AppendLine('')
  [void]$b.AppendLine('Roster:')
  foreach ($r in $roster.roles) { [void]$b.AppendLine("  $($r.key)  $($r.name)") }
  [void]$b.AppendLine('')
  try {
    $issues = Get-SwarmIssues $QueueJsonFile
    [void]$b.AppendLine('Open swarm issues by status:')
    $c = Get-QueueCounts $issues
    foreach ($k in $c.Keys) { [void]$b.AppendLine("  status:$k = $($c[$k])") }
    [void]$b.AppendLine('')
    [void]$b.AppendLine('Titles:')
    foreach ($i in $issues) { $st = (Get-LabelNames $i | Where-Object { $_ -like 'status:*' } | Select-Object -First 1); [void]$b.AppendLine("  #$($i.number) [$st] $($i.title)") }
  } catch { [void]$b.AppendLine("Issue queue unavailable: $($_.Exception.Message)") }
  Remove-Secrets $b.ToString()
}
try {
  $remote = $PushRemote
  $snap = New-Snapshot
  if ($WhatIfPreference) { Write-Host "What if: would push swarm-snapshot.txt to branch swarm-state of $remote"; Write-Host $snap; exit 0 }
  $tmp = Join-Path ([IO.Path]::GetTempPath()) ("swarm-snap-" + [guid]::NewGuid())
  New-Item -ItemType Directory -Path $tmp | Out-Null
  $ErrorActionPreference = 'Continue'   # git writes progress to stderr; exit codes are checked explicitly
  try {
    function G { & git -C $tmp -c user.name='swarm-backup' -c user.email='swarm-backup@users.noreply.github.com' @gitSafe @args 2>&1 | Out-Null; if ($LASTEXITCODE -ne 0) { throw "git $($args[0]) failed" } }
    G init -q
    G remote add origin $remote
    & git -C $tmp @gitSafe fetch -q --depth 1 origin swarm-state 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) { G checkout -q -B swarm-state FETCH_HEAD } else { G checkout -q --orphan swarm-state }
    $snapFile = Join-Path $tmp 'swarm-snapshot.txt'
    if (Test-Path -LiteralPath $snapFile) {   # unchanged apart from the timestamp line: no commit, no push
      $body = { param($t) (($t -replace "`r", '') -split "`n" | Select-Object -Skip 1 | ForEach-Object { $_.TrimEnd() }) -join "`n" }
      if ((& $body (Get-Content -Raw -LiteralPath $snapFile)).Trim() -eq (& $body $snap).Trim()) { Write-Host 'snapshot unchanged, not pushing'; exit 0 }
    }
    Set-Content -LiteralPath $snapFile -Value $snap -Encoding UTF8
    G add swarm-snapshot.txt
    G commit -q -m "swarm snapshot $($NowUtc.ToString('yyyy-MM-ddTHH:mm:ssZ'))"
    G push -q origin swarm-state            # plain push, never forced
    Write-Host 'snapshot pushed to branch swarm-state'
  } finally { Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue }
} catch { Write-Warning "snapshot push skipped (backup itself succeeded): $_" }
exit 0
