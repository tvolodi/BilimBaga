<#
 Dry-run tests for up.ps1 / ensure-up.ps1 / down.ps1 / install-ensure-task.ps1.
 Never launches, stops or registers anything: every script runs with -WhatIf, `claude agents --json` is replaced
 by a stub file, and SWARM_STATE_DIR points at a temp directory.
 Usage: powershell -File swarm\tests\test-ensure-up.ps1
#>
$ErrorActionPreference = 'Stop'
$swarm = Split-Path -Parent $PSScriptRoot
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("swarm-test-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
$env:SWARM_STATE_DIR = Join-Path $tmp 'state'
New-Item -ItemType Directory -Path $env:SWARM_STATE_DIR | Out-Null
$fail = 0
function Check([string]$name, [bool]$ok) { if ($ok) { Write-Host "PASS $name" } else { Write-Host "FAIL $name"; $script:fail++ } }
function Run([string]$script, [string[]]$more) { (& powershell.exe -NoProfile -File (Join-Path $swarm $script) @more 2>&1 | Out-String) }
function Stub($agents) { $f = Join-Path $tmp ("agents-" + [guid]::NewGuid() + ".json"); ConvertTo-Json -InputObject @($agents) -Depth 4 | Set-Content $f; $f }

# syntax check of every script
foreach ($s in 'lib.ps1','up.ps1','down.ps1','ensure-up.ps1','install-ensure-task.ps1','bin\record-session.ps1') {
  $err = $null; [void][System.Management.Automation.Language.Parser]::ParseFile((Join-Path $swarm $s), [ref]$null, [ref]$err)
  Check "syntax $s" (-not $err)
}

$sidDev1 = '11111111-1111-1111-1111-111111111111'
$sidInfra = '22222222-2222-2222-2222-222222222222'

# 1. nothing live, no saved ids: ensure-up relaunches every bb-* role except never-launched bb-infra, all fresh
$none = Stub @()
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $none)
Check 'ensure-up relaunches dev1' ($o -match 'relaunching dev1')
Check 'ensure-up skips never-launched bb-infra' ($o -notmatch 'relaunching infra')
Check 'ensure-up WhatIf writes no state' (-not (Get-ChildItem $env:SWARM_STATE_DIR))

# 2. saved id => --resume with same name/settings and env cleaning; fallback present
Set-Content (Join-Path $env:SWARM_STATE_DIR 'sessions.json') ('{"dev1":{"session_id":"' + $sidDev1 + '","name":"bb-dev1"}}')
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $none)
Check 'dev1 resumed with saved id' ($o -match "claude --resume $sidDev1 -n bb-dev1 --permission-mode bypassPermissions --settings '[^']*dev1\.settings\.json'")
Check 'env cleaning present' ($o -match 'CLAUDE_CODE_\(CHILD_SESSION')
Check 'startup preamble in resume prompt' ($o -match 'Re-read your checkpoint file')
Check 'other roles fresh' ($o -match 'claude --session-id')

# 3. live bb-dev1 is not touched; ai-dala-infra-fc live => never touched, bb-infra not started
$live = Stub @(@{ name = 'bb-dev1'; pid = 4001 }, @{ name = 'ai-dala-infra-fc'; pid = 4002 })
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $live)
Check 'live dev1 not relaunched' ($o -notmatch 'relaunching dev1')
Check 'infra untouched while ai-dala-infra live' (($o -notmatch 'relaunching infra') -and ($o -notmatch 'ai-dala-infra'))

# 4. bb-infra launched earlier + ai-dala-infra gone => resumed
Set-Content (Join-Path $env:SWARM_STATE_DIR 'sessions.json') ('{"infra":{"session_id":"' + $sidInfra + '","name":"bb-infra"}}')
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $none)
Check 'bb-infra resumed when previously launched' ($o -match "claude --resume $sidInfra -n bb-infra")

# 5. supervisor watchdog
$all = Stub @(@{ name = 'bb-dev1' }, @{ name = 'bb-dev2' }, @{ name = 'bb-ba' }, @{ name = 'bb-uat' }, @{ name = 'ai-dala-infra-fc' }, @{ name = 'bb-supervisor'; pid = 4003 })
$now = [datetime]'2026-10-09T12:00:00Z'
Set-Content (Join-Path $env:SWARM_STATE_DIR 'supervisor.json') '{"role":"supervisor","last_tick_utc":"2026-10-09T10:30:00Z"}'
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $all, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'stale supervisor is nudged first' ($o -match 'nudging' -and $o -notmatch 'relaunching supervisor')
Set-Content (Join-Path $env:SWARM_STATE_DIR 'supervisor.nudge.json') '{"nudged_utc":"2026-10-09T11:40:00Z"}'
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $all, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'stale after grace: stop+relaunch resumed' ($o -match 'restarting it resumed' -and $o -match 'relaunching supervisor')
Set-Content (Join-Path $env:SWARM_STATE_DIR 'supervisor.nudge.json') '{"nudged_utc":"2026-10-09T11:55:00Z"}'
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $all, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'within grace: only waits' ($o -match 'within grace' -and $o -notmatch 'relaunching supervisor')
Set-Content (Join-Path $env:SWARM_STATE_DIR 'supervisor.json') '{"role":"supervisor","last_tick_utc":"2026-10-09T11:50:00Z"}'
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $all, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'fresh heartbeat: nothing' ($o -notmatch 'nudging' -and $o -notmatch 'relaunching')
Check 'WhatIf left nudge file untouched' ((Get-Content (Join-Path $env:SWARM_STATE_DIR 'supervisor.nudge.json') -Raw) -match '11:55:00')

# 6. up.ps1 dry run
$o = Run 'up.ps1' @('-WhatIf', '-AgentsJsonFile', $live, '-Roles', 'dev1,dev2,infra')
Check 'up skips running dev1' ($o -match 'dev1 already running')
Check 'up reuses ai-dala-infra' ($o -match 'infra: reusing running session \(ai-dala-infra-fc\)')
Check 'up launches dev2' ($o -match 'launching dev2')

. (Join-Path $swarm 'lib.ps1')
$ro = Get-Roster
$mainCo = Get-MainCheckout $ro
Check 'main checkout resolved (has .git directory)' (Test-Path (Join-Path $mainCo '.git') -PathType Container)
Check 'main checkout is not a nested worktree path' ($mainCo -notmatch '\.claude[\\/]worktrees')
$sess = Read-Sessions $env:SWARM_STATE_DIR
$bad = 0
foreach ($r in $ro.roles) {
  foreach ($fresh in $true, $false) {
    $sp = New-LaunchSpec $r $ro $env:SWARM_STATE_DIR @{ ($r.key) = [pscustomobject]@{ session_id = $sidDev1 } } -Fresh:$fresh
    $e = $null; [void][System.Management.Automation.Language.Parser]::ParseInput($sp.Command, [ref]$null, [ref]$e)
    if ($e) { $bad++; Write-Host "  parse error in $($r.key) fresh=$fresh : $($e[0].Message)" }
  }
}
Check 'generated launch commands are valid PowerShell' ($bad -eq 0)
$sp = New-LaunchSpec $ro.roles[0] $ro $env:SWARM_STATE_DIR @{ dev1 = [pscustomobject]@{ session_id = $sidDev1 } }
Check 'fresh fallback only when resume dies within 20 s' ($sp.Command -match 'TotalSeconds -lt 20')

# 7. down.ps1 dry run: stops bb-* only, not infra
$o = Run 'down.ps1' @('-WhatIf', '-AgentsJsonFile', $live)
Check 'down stops bb-dev1' ($o -match 'bb-dev1')
Check 'down leaves ai-dala-infra-fc' ($o -notmatch 'ai-dala-infra')
Check 'down handles task step' ($o -match 'scheduled task')

# 8. installer dry run
$o = Run 'install-ensure-task.ps1' @('-WhatIf')
Check 'installer previews trigger' ($o -match 'at logon, repeat every 5 min' -and $o -match 'What if')
Check 'task not registered' (-not (Get-ScheduledTask -TaskName 'BilimBaga-Swarm-EnsureUp' -ErrorAction SilentlyContinue))

# 9. missing live listing => do nothing
$bad = Join-Path $tmp 'bad.json'; Set-Content $bad ''
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $bad)
Check 'empty agent list is treated as no live sessions (not an error)' ($o -match 'relaunching')

Remove-Item -Recurse -Force $tmp
Remove-Item Env:SWARM_STATE_DIR
if ($fail) { Write-Host "$fail FAILED"; exit 1 } else { Write-Host 'ALL PASSED' }
