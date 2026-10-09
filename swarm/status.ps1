<#
 One-page swarm status (read-only): roster, live session status, each worker's checkpoint with heartbeat age,
 and open issue counts by status label. Same fail-safe as ensure-up: an unrecognised `claude agents --json`
 is reported as "unknown", never as "all sessions down".
 Usage: powershell -File swarm\status.ps1 [-AgentsJsonFile stub.json] [-QueueJsonFile stub.json] [-NowUtc <datetime>]
#>
[CmdletBinding()]
param(
  [string]$AgentsJsonFile,
  [string]$QueueJsonFile,
  [datetime]$NowUtc = ([datetime]::UtcNow)
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'lib.ps1')
$roster = Get-Roster
$stateDir = Get-StateDir $roster
$NowUtc = $NowUtc.ToUniversalTime()
$staleMin = [int]$roster.stale_supervisor_min

$live = $null; $liveErr = $null
try { $live = Get-RecognisedLiveAgents $AgentsJsonFile } catch { $liveErr = "$($_.Exception.Message)" }

Write-Host "== BilimBaga swarm status  $($NowUtc.ToString('yyyy-MM-dd HH:mm:ss'))Z =="
Write-Host ''
Write-Host '-- Roster / sessions --'
if ($liveErr) { Write-Host "  (live session list unavailable: $liveErr)" }
foreach ($r in $roster.roles) {
  $state = 'unknown'
  $extra = ''
  if ($live) {
    $a = Find-LiveAgent $live $r.name
    if ($a) {
      $state = 'LIVE'
      $st = if ($a.status) { $a.status } elseif ($a.state) { $a.state } else { $null }
      if ($st -is [string]) { $extra = " ($st)" }
    } else { $state = 'not running' }
  }
  Write-Host ("  {0,-11} {1,-14} {2}{3}" -f $r.key, $r.name, $state, $extra)
}
Write-Host ''
Write-Host '-- Checkpoints (heartbeat age) --'
foreach ($r in $roster.roles) {
  $c = Get-Checkpoint $stateDir $r.key $NowUtc
  if (-not $c) { Write-Host ("  {0,-11} no checkpoint" -f $r.key); continue }
  if ($c.unreadable) { Write-Host ("  {0,-11} checkpoint unreadable" -f $r.key); continue }
  $age = if ($null -ne $c.age_min) { '{0:N0} min ago{1}' -f $c.age_min, $(if ($c.age_min -gt $staleMin) { ' STALE' } else { '' }) } else { 'heartbeat unknown' }
  Write-Host ("  {0,-11} {1,-7} issue={2} step={3} [{4}]" -f $r.key, $c.status, $c.issue, $c.step, $age)
  if ($c.next_action) { Write-Host "              next: $($c.next_action)" }
  if ($c.blockers) { Write-Host "              blockers: $($c.blockers)" }
}
Write-Host ''
Write-Host '-- Issue queue (open, label swarm) --'
try {
  $issues = Get-SwarmIssues $QueueJsonFile
  $counts = Get-QueueCounts $issues
  Write-Host ('  ' + (($counts.Keys | ForEach-Object { "$_=$($counts[$_])" }) -join '  '))
  Write-Host "  total open: $($issues.Count)"
} catch { Write-Host "  (queue unavailable: $($_.Exception.Message))" }
