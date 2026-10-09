<#
 Dry-run tests for backup-state.ps1 and status.ps1. Everything happens in a temp dir: BackupRoot is a temp
 folder (never ~/.swarm-backups), state is SWARM_STATE_DIR, `claude agents --json` / `gh issue list` are stub
 files, and the snapshot push goes to a temp BARE repo (never GitHub). Nothing is launched, stopped or registered.
 Usage: powershell -File swarm\tests\test-backup-status.ps1
#>
$ErrorActionPreference = 'Continue'
$swarm = Split-Path -Parent $PSScriptRoot
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("swarm-btest-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
$env:SWARM_STATE_DIR = Join-Path $tmp 'state'
New-Item -ItemType Directory -Path $env:SWARM_STATE_DIR | Out-Null
$fail = 0
function Check([string]$name, [bool]$ok) { if ($ok) { Write-Host "PASS $name" } else { Write-Host "FAIL $name"; $script:fail++ } }
function Run([string]$script, [string[]]$more) { (& powershell.exe -NoProfile -File (Join-Path $swarm $script) @more 2>&1 | Out-String) }
function Json([string]$name, $obj) { $f = Join-Path $tmp $name; ConvertTo-Json -InputObject $obj -Depth 5 | Set-Content $f; $f }

foreach ($s in 'backup-state.ps1', 'status.ps1', 'lib.ps1') {
  $err = $null; [void][System.Management.Automation.Language.Parser]::ParseFile((Join-Path $swarm $s), [ref]$null, [ref]$err)
  Check "syntax $s" (-not $err)
}

$queue = Json 'queue.json' @(
  @{ number = 1; title = 'first'; labels = @(@{ name = 'swarm' }, @{ name = 'status:ready' }) },
  @{ number = 2; title = 'leaky ghp_abcdefghijklmnopqrstuvwxyz0123456789'; labels = @(@{ name = 'swarm' }, @{ name = 'status:in-progress' }) },
  @{ number = 3; title = 'third password=hunter2'; labels = @(@{ name = 'swarm' }, @{ name = 'status:ready' }) })

# checkpoint + session files in the temp state dir
Set-Content (Join-Path $env:SWARM_STATE_DIR 'dev1.json') '{"role":"dev1","status":"busy","issue":149,"branch":"b","step":"tests","next_action":"open PR","last_tick_utc":"2026-10-09T11:50:00Z","blockers":""}'
Set-Content (Join-Path $env:SWARM_STATE_DIR 'supervisor.json') '{"role":"supervisor","status":"idle","issue":null,"last_tick_utc":"2026-10-09T09:00:00Z"}'
Set-Content (Join-Path $env:SWARM_STATE_DIR 'sessions.json') '{"dev1":{"session_id":"11111111-1111-1111-1111-111111111111"}}'

# --- backup: WhatIf writes nothing ---
$root = Join-Path $tmp 'backups'
$o = Run 'backup-state.ps1' @('-WhatIf', '-BackupRoot', $root, '-QueueJsonFile', $queue, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'backup WhatIf plans the folder' ($o -match '20261009-120000')
Check 'backup WhatIf creates nothing' (-not (Test-Path $root))
Check 'backup WhatIf snapshot is scrubbed' (($o -match 'REDACTED') -and ($o -notmatch 'ghp_abcdef') -and ($o -notmatch 'hunter2'))
Check 'snapshot excludes session ids' ($o -notmatch '11111111-1111')

# --- real copy into temp root, with foreign content that must survive pruning ---
New-Item -ItemType Directory -Path $root | Out-Null
Set-Content (Join-Path $root 'keep-me.txt') 'x'
New-Item -ItemType Directory -Path (Join-Path $root 'notes-2026') | Out-Null
New-Item -ItemType Directory -Path (Join-Path $root '20261009-1200') | Out-Null       # near miss: wrong pattern
New-Item -ItemType Directory -Path (Join-Path $root '2026100a-120000') | Out-Null     # near miss
1..22 | ForEach-Object { New-Item -ItemType Directory -Path (Join-Path $root ('20260901-{0:D6}' -f $_)) | Out-Null }
$o = Run 'backup-state.ps1' @('-NoPush', '-BackupRoot', $root, '-NowUtc', '2026-10-09T12:00:00Z')
$d = Join-Path $root '20261009-120000'
Check 'backup copied state json' (Test-Path (Join-Path $d 'state\dev1.json'))
Check 'backup copied roster' (Test-Path (Join-Path $d 'roster.json'))
Check 'backup copied role settings' (Test-Path (Join-Path $d 'roles\dev1.settings.json'))
$dated = @(Get-ChildItem $root -Directory | Where-Object { $_.Name -match '^\d{8}-\d{6}$' })
Check 'pruned to the newest 20 dated folders' ($dated.Count -eq 20)
Check 'newest folder kept, oldest pruned' ((Test-Path $d) -and -not (Test-Path (Join-Path $root '20260901-000001')))
Check 'foreign files and near-miss names untouched' ((Test-Path (Join-Path $root 'keep-me.txt')) -and (Test-Path (Join-Path $root 'notes-2026')) -and (Test-Path (Join-Path $root '20261009-1200')) -and (Test-Path (Join-Path $root '2026100a-120000')))

# --- throttle ---
$o = Run 'backup-state.ps1' @('-NoPush', '-BackupRoot', $root, '-MinIntervalMin', '60', '-NowUtc', ([datetime]::UtcNow.ToString('o')))
Check 'min-interval skips a recent backup' ($o -match 'skipping')

# --- snapshot push to a temp bare repo; failure does not fail the backup ---
$bare = Join-Path $tmp 'remote.git'
& git init -q --bare $bare 2>&1 | Out-Null
$root2 = Join-Path $tmp 'backups2'
$o = Run 'backup-state.ps1' @('-BackupRoot', $root2, '-PushRemote', $bare, '-QueueJsonFile', $queue, '-NowUtc', '2026-10-09T12:00:00Z')
$txt = (& git -C $bare show swarm-state:swarm-snapshot.txt 2>&1 | Out-String)
Check 'snapshot pushed to branch swarm-state' ($txt -match 'status:ready = 2' -and $txt -match 'bb-dev1')
Check 'pushed snapshot has no secrets' (($txt -notmatch 'ghp_abcdef') -and ($txt -notmatch 'hunter2') -and ($txt -notmatch 'session_id'))
$files = (& git -C $bare ls-tree -r --name-only swarm-state 2>&1 | Out-String).Trim()
Check 'branch holds only the snapshot file' ($files -eq 'swarm-snapshot.txt')
$o = Run 'backup-state.ps1' @('-BackupRoot', $root2, '-PushRemote', $bare, '-QueueJsonFile', $queue, '-NowUtc', '2026-10-09T13:00:00Z')
Check 'second push is a fast-forward (2 commits)' ((& git -C $bare rev-list --count swarm-state) -eq '2')
$o = Run 'backup-state.ps1' @('-BackupRoot', $root2, '-PushRemote', (Join-Path $tmp 'no-such.git'), '-QueueJsonFile', $queue, '-NowUtc', '2026-10-09T14:00:00Z')
Check 'failed push only warns, backup still made' (($o -match 'snapshot push skipped') -and (Test-Path (Join-Path $root2 '20261009-140000\roster.json')) -and ($LASTEXITCODE -eq 0))

# --- status.ps1 ---
$agents = Json 'agents.json' @(@{ name = 'bb-dev1'; pid = 1; status = 'busy' }, @{ name = 'bb-supervisor'; pid = 2 })
$o = Run 'status.ps1' @('-AgentsJsonFile', $agents, '-QueueJsonFile', $queue, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'status shows live and missing sessions' ($o -match 'bb-dev1\s+LIVE \(busy\)' -and $o -match 'bb-dev2\s+not running')
Check 'status shows checkpoint with heartbeat age' ($o -match 'dev1\s+busy\s+issue=149 step=tests \[10 min ago\]')
Check 'status flags stale heartbeat' ($o -match 'supervisor.*180 min ago STALE')
Check 'status shows missing checkpoint' ($o -match 'dev2\s+no checkpoint')
Check 'status shows queue counts' ($o -match 'ready=2\s+in-progress=1' -and $o -match 'total open: 3')

# fail-safe: unrecognised agents schema => unknown, never "not running"; queue failure degrades
$bad = Json 'bad.json' @(@{ title = 'x' })
$o = Run 'status.ps1' @('-AgentsJsonFile', $bad, '-QueueJsonFile', (Join-Path $tmp 'missing.json'))
Check 'unrecognised schema reported unknown, not down' (($o -match 'unavailable') -and ($o -match 'dev1\s+bb-dev1\s+unknown') -and ($o -notmatch 'not running'))
Check 'queue failure degrades gracefully' ($o -match 'queue unavailable')

# ensure-up still dry-runs cleanly and does not back up in dry-run
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $agents)
Check 'ensure-up skips backup in dry run' ($o -match 'backup: skipped')

Remove-Item -Recurse -Force $tmp
Remove-Item Env:SWARM_STATE_DIR
if ($fail) { Write-Host "$fail FAILED"; exit 1 } else { Write-Host 'ALL PASSED' }
