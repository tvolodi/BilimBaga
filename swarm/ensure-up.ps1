<#
 Dumb liveness check (no decisions about work). Reads `claude agents --json`; any roster role whose
 bb-* session is missing is relaunched RESUMED (--resume <saved id>, same -n name and settings, same
 env cleaning as up.ps1) in a Windows Terminal tab. Run by the user-level Scheduled Task installed by
 install-ensure-task.ps1 (at logon, then every 5 minutes).
 - Never touches ai-dala-infra*: only roster names starting with 'bb-' are considered, and bb-infra is
   considered only if it was launched before (has a saved session id) and no ai-dala-infra* session is live.
 - Supervisor watchdog: if swarm/state/supervisor.json last_tick_utc is older than stale_supervisor_min
   (45) while bb-supervisor is live, nudge it first (marker file swarm/state/supervisor.nudge.json, which the
   Supervisor reads each tick; no CLI exists for sending messages); supervisor_grace_min (10) later, if
   still stale, stop that session and relaunch it resumed.
 - FAIL-SAFE: unless `claude agents --json` is a JSON array with at least one entry having a string `name`, the
   run does nothing at all (no relaunch, no stop): failure, empty output, null, object, [], other field names.
 - Stop only when ALL hold: heartbeat stale, a nudge exists, nudge older than supervisor_grace_min, and the heartbeat
   re-read immediately before stopping is STILL stale. A fresh heartbeat clears the nudge and never stops.
 Usage: powershell -File swarm\ensure-up.ps1 [-WhatIf] [-AgentsJsonFile stub.json] [-NowUtc <datetime>]
#>
[CmdletBinding(SupportsShouldProcess)]
param(
  [string]$AgentsJsonFile,          # test stub replacing `claude agents --json`
  [datetime]$NowUtc = ([datetime]::UtcNow)
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'lib.ps1')
$roster   = Get-Roster
$stateDir = Get-StateDir $roster
$staleMin = [int]$roster.stale_supervisor_min
$graceMin = [int]$roster.supervisor_grace_min
$NowUtc   = $NowUtc.ToUniversalTime()

function Start-Resumed($r) {
  # cooldown: a role slow to register must not be relaunched again by the next 5-min run
  $cdFile = Join-Path $stateDir 'ensure-up.relaunch.json'
  $cd = @{}
  if (Test-Path -LiteralPath $cdFile) { try { (Get-Content -Raw -LiteralPath $cdFile | ConvertFrom-Json).PSObject.Properties | ForEach-Object { $cd[$_.Name] = $_.Value } } catch {} }
  if ($cd[$r.key] -and ($NowUtc - ([datetime]$cd[$r.key]).ToUniversalTime()).TotalMinutes -lt 8) { Write-Host "$($r.key): relaunched less than 8 min ago, waiting"; return }
  $spec = New-LaunchSpec $r $roster $stateDir $script:sessions
  Write-Host "relaunching $($r.key) ($($r.name)) mode=$($spec.Mode) session=$($spec.SessionId)"
  if ($WhatIfPreference) { Write-Host "  cwd: $(Expand-Path $r.cwd $roster)"; Write-Host "  cmd: $($spec.Command)" }
  if ($PSCmdlet.ShouldProcess($r.name, "relaunch $($spec.Mode)")) {
    if ($spec.Mode -eq 'fresh') { Save-Session $stateDir $r.key $r.name $spec.SessionId }
    Start-RoleTab $r $spec $roster $script:haveWt
    $cd[$r.key] = $NowUtc.ToString('yyyy-MM-ddTHH:mm:ssZ')
    ($cd | ConvertTo-Json) | Set-Content -LiteralPath $cdFile -Encoding UTF8
  }
}

function Get-HeartbeatAgeMin {
  $hb = Join-Path $stateDir 'supervisor.json'
  if (-not (Test-Path -LiteralPath $hb)) { return $null }
  try { $t = ([datetime](Get-Content -Raw -LiteralPath $hb | ConvertFrom-Json).last_tick_utc).ToUniversalTime() } catch { $t = (Get-Item -LiteralPath $hb).LastWriteTimeUtc }  # unparsable: file mtime
  ($NowUtc - $t).TotalMinutes
}

# Hourly state backup: runs AFTER the mutex is released, in a child process with a 120 s hard timeout
# (tree-killed), so a hung git/gh call can never block later watchdog runs. Marker-throttled to once per hour
# even when it times out. Never fails or delays the liveness check; skipped for dry runs/stubs.
function Invoke-Backup {
  if ($WhatIfPreference -or $AgentsJsonFile) { Write-Host 'backup: skipped (dry run)'; return }
  [void](Invoke-BoundedBackup (Join-Path $PSScriptRoot 'backup-state.ps1') @('-MinIntervalMin', '60') (Join-Path $stateDir 'backup.last.json') $NowUtc 120 60)
}

# one run at a time (the task fires every 5 min; launching tabs can take a while)
$mutex = New-Object System.Threading.Mutex($false, 'Global\bilimbaga-swarm-ensure-up')
if (-not $mutex.WaitOne(0)) { Write-Host 'another ensure-up run is active, exiting'; exit 0 }
$liveFailed = $false
try {
  try { $live = Get-RecognisedLiveAgents $AgentsJsonFile } catch { Write-Warning "cannot get a recognised live-session list, doing nothing (no relaunch, no stop): $_"; $liveFailed = $true }
  if (-not $liveFailed) {
  if (-not $WhatIfPreference) { Update-SessionsFromLive $roster $live $stateDir }
  $script:sessions = Read-Sessions $stateDir
  $script:haveWt = [bool](Get-Command wt.exe -ErrorAction SilentlyContinue)

  # 1. missing roles
  foreach ($r in $roster.roles) {
    if ($r.name -notlike 'bb-*') { continue }                                   # never anything else
    if (Find-LiveAgent $live $r.name) { continue }
    if ($r.reuse_live_prefix) {
      if (Find-LiveAgent $live $r.reuse_live_prefix) { continue }                # external infra session is alive
      if (-not (Get-SavedSessionId $script:sessions $r.key)) { continue }        # bb-infra never launched: not ours to start
    }
    Start-Resumed $r
  }

  # 2. Supervisor watchdog
  $sup = $roster.roles | Where-Object { $_.key -eq 'supervisor' }
  $supLive = if ($sup) { Find-LiveAgent $live $sup.name }
  $nudgeFile = Join-Path $stateDir 'supervisor.nudge.json'
  if ($supLive) {
    $ageMin = Get-HeartbeatAgeMin
    if ($null -ne $ageMin -and $ageMin -gt $staleMin) {
      $nudge = $null
      if (Test-Path -LiteralPath $nudgeFile) { try { $nudge = ([datetime](Get-Content -Raw -LiteralPath $nudgeFile | ConvertFrom-Json).nudged_utc).ToUniversalTime() } catch {} }
      if (-not $nudge) {
        Write-Host ("supervisor heartbeat stale ({0:N0} min): nudging" -f $ageMin)
        if ($PSCmdlet.ShouldProcess($sup.name, 'nudge (write supervisor.nudge.json)')) {
          if (-not (Test-Path -LiteralPath $stateDir)) { New-Item -ItemType Directory -Path $stateDir -Force | Out-Null }
          $msg = '{"nudged_utc":"' + $NowUtc.ToString('yyyy-MM-ddTHH:mm:ssZ') + '","reason":"supervisor.json last_tick_utc older than ' + $staleMin + ' min; run a tick and refresh your heartbeat"}'
          Set-Content -LiteralPath $nudgeFile -Value $msg -Encoding UTF8
        }
      } elseif (($NowUtc - $nudge).TotalMinutes -le $graceMin) {
        Write-Host 'supervisor nudged, within grace period'
      } else {
        # last look at the heartbeat: it may have been refreshed since the check above
        $age2 = Get-HeartbeatAgeMin
        if ($null -eq $age2 -or $age2 -le $staleMin -or -not $supLive.pid) {
          Write-Host 'supervisor heartbeat refreshed (or no pid), not stopping; clearing nudge'
          if ($PSCmdlet.ShouldProcess($nudgeFile, 'remove')) { Remove-Item -LiteralPath $nudgeFile -Force -ErrorAction SilentlyContinue }
        } else {
        Write-Host "supervisor still stale $graceMin min after nudge: restarting it resumed"
        if ($PSCmdlet.ShouldProcess($sup.name, "stop pid $($supLive.pid) and relaunch resumed")) {
          if ($supLive.pid) { Stop-Process -Id $supLive.pid -Force -ErrorAction SilentlyContinue; Wait-Process -Id $supLive.pid -Timeout 15 -ErrorAction SilentlyContinue }
          $script:sessions = Read-Sessions $stateDir
          Remove-Item -LiteralPath $nudgeFile -Force -ErrorAction SilentlyContinue
        }
        Start-Resumed $sup
        }
      }
    } elseif (Test-Path -LiteralPath $nudgeFile) {
      Write-Host 'supervisor heartbeat fresh again, clearing nudge'
      if ($PSCmdlet.ShouldProcess($nudgeFile, 'remove')) { Remove-Item -LiteralPath $nudgeFile -Force }
    }
  }
  }
} finally { $mutex.ReleaseMutex(); $mutex.Dispose() }
Invoke-Backup   # mutex released: a slow backup cannot block the next run
if ($liveFailed) { exit 1 }
