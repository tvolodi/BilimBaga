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
$none = Stub @(@{ name = 'unrelated-session'; pid = 1 })   # valid recognised list lacking every bb-* role
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

# 5. supervisor watchdog (all under -WhatIf unless stated; planned actions are asserted from output)
$all = Stub @(@{ name = 'bb-dev1' }, @{ name = 'bb-dev2' }, @{ name = 'bb-ba' }, @{ name = 'bb-uat' }, @{ name = 'ai-dala-infra-fc' }, @{ name = 'bb-supervisor'; pid = 4003 })
$sd = $env:SWARM_STATE_DIR
$hbF = Join-Path $sd 'supervisor.json'; $ndF = Join-Path $sd 'supervisor.nudge.json'
function Wd([string]$now = '2026-10-09T12:00:00Z', [string[]]$extra = @('-WhatIf')) { Run 'ensure-up.ps1' (@('-AgentsJsonFile', $all, '-NowUtc', $now) + $extra) }
function Heartbeat([string]$t) { Set-Content $hbF ('{"role":"supervisor","last_tick_utc":"' + $t + '"}') }
function Nudge([string]$t) { Set-Content $ndF ('{"nudged_utc":"' + $t + '"}') }
function NoStop([string]$o) { ($o -notmatch 'stop pid') -and ($o -notmatch 'restarting it') -and ($o -notmatch 'relaunching') }
Remove-Item $ndF -ErrorAction SilentlyContinue

Heartbeat '2026-10-09T11:50:00Z'   # fresh
$o = Wd
Check 'W1 fresh heartbeat, no nudge: no nudge, no stop' ($o -notmatch 'nudging' -and (NoStop $o))

Heartbeat '2026-10-09T10:30:00Z'   # 90 min stale
$o = Wd
Check 'W2 stale, no nudge file: nudge only, no stop' ($o -match 'nudging' -and $o -match 'What if.*nudge' -and (NoStop $o))
Check 'W2 WhatIf wrote no nudge file' (-not (Test-Path $ndF))

Nudge '2026-10-09T11:55:00Z'       # 5 min ago
$o = Wd
Check 'W3 stale + nudge 5 min old: waits, no stop' ($o -match 'within grace' -and (NoStop $o))
Nudge '2026-10-09T11:50:00Z'       # exactly 10 min: not yet past cooldown
$o = Wd
Check 'W3b stale + nudge exactly 10 min old: no stop' ($o -match 'within grace' -and (NoStop $o))

Nudge '2026-10-09T11:40:00Z'       # 20 min ago, heartbeat still stale
$o = Wd
Check 'W4 stale + nudge >=10 min: stop planned (pid 4003) and relaunch resumed' ($o -match 'What if.*stop pid 4003' -and $o -match 'restarting it resumed' -and $o -match 'relaunching supervisor')
Check 'W4 WhatIf did not touch nudge file' ((Get-Content $ndF -Raw) -match '11:40:00')

# heartbeat refreshed after the nudge was written: no stop, nudge cleared (real run is safe: every role is live => nothing is launched)
Heartbeat '2026-10-09T11:58:00Z'
$o = Wd
Check 'W5 refreshed heartbeat + old nudge (WhatIf): no stop, clearing planned' ($o -match 'fresh again' -and $o -match 'What if.*remove' -and (NoStop $o))
$o = Wd '2026-10-09T12:00:00Z' @()
Check 'W5 refreshed heartbeat + old nudge (real run): no stop/relaunch' (NoStop $o)
Check 'W5 nudge cleared' (-not (Test-Path $ndF))

# stale again later: a new nudge is needed first (a cleared nudge must not carry over into a stop)
Heartbeat '2026-10-09T10:30:00Z'
$o = Wd
Check 'W6 stale again after clear: nudge first, no stop' ($o -match 'nudging' -and (NoStop $o))

# 8-minute relaunch cooldown (cooldown file written by hand; WhatIf never writes it)
Remove-Item $ndF, $hbF -ErrorAction SilentlyContinue
$cdF = Join-Path $sd 'ensure-up.relaunch.json'
Set-Content $cdF '{"dev1":"2026-10-09T11:55:00Z"}'            # 5 min ago
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $none, '-NowUtc', '2026-10-09T12:00:00Z')
Check 'C1 relaunched 5 min ago: dev1 waits' ($o -match 'dev1: relaunched less than 8 min ago' -and $o -notmatch 'relaunching dev1')
Check 'C1 other roles unaffected by dev1 cooldown' ($o -match 'relaunching dev2')
$o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $none, '-NowUtc', '2026-10-09T12:04:00Z')   # 9 min after
Check 'C2 relaunched 9 min ago: dev1 relaunched again' ($o -match 'relaunching dev1')
Remove-Item $cdF -ErrorAction SilentlyContinue

# FAIL-SAFE: unrecognised / empty / failed agent list => zero relaunch and zero stop, even with a stale supervisor + old nudge
Heartbeat '2026-10-09T10:30:00Z'; Nudge '2026-10-09T11:00:00Z'
Set-Content (Join-Path $sd 'sessions.json') ('{"dev1":{"session_id":"' + $sidDev1 + '","name":"bb-dev1"}}')
$cases = [ordered]@{
  'empty array'                 = '[]'
  'null'                        = 'null'
  'empty file'                  = ''
  'non-JSON'                    = 'Error: not logged in'
  'object instead of array'     = '{"agents":[{"name":"bb-dev1"}]}'
  'array of scalars'            = '["bb-dev1","bb-dev2"]'
  'other field names'           = '[{"title":"bb-dev1","pid":1},{"title":"bb-supervisor","pid":2}]'
  'name not a string'           = '[{"name":42},{"name":null}]'
  'array of nulls'              = '[null]'
}
foreach ($k in $cases.Keys) {
  $f = Join-Path $tmp ("failsafe-" + [guid]::NewGuid() + ".json"); Set-Content $f $cases[$k]
  $o = Run 'ensure-up.ps1' @('-WhatIf', '-AgentsJsonFile', $f, '-NowUtc', '2026-10-09T12:00:00Z')
  Check "F fail-safe ($k): does nothing" (($o -match 'doing nothing') -and ($o -notmatch 'relaunching') -and ($o -notmatch 'stop pid') -and ($o -notmatch 'nudging') -and ($o -notmatch 'What if'))
  $o = Run 'ensure-up.ps1' @('-AgentsJsonFile', $f, '-NowUtc', '2026-10-09T12:00:00Z')   # real run is safe: it exits before any action
  Check "F fail-safe ($k, real run): no launch, no state change" (($o -match 'doing nothing') -and ((Get-Content $ndF -Raw) -match '11:00:00') -and -not (Test-Path (Join-Path $sd 'ensure-up.relaunch.json')))
}
# failing command (not a file): cannot be simulated without running claude; covered by the throw on $LASTEXITCODE in lib.ps1
Remove-Item $ndF, $hbF, (Join-Path $sd 'sessions.json') -ErrorAction SilentlyContinue

# 6. up.ps1 dry run. up.ps1 is the minimal launcher from #285 (no -WhatIf / -AgentsJsonFile), so the wrapper
#    shadows `claude` (live list comes from the stub file) and Start-Process (prints instead of opening a tab).
$upWrap = Join-Path $tmp 'up-dryrun.ps1'
Set-Content $upWrap @"
param([string]`$Stub, [string]`$Up)
function claude { Get-Content -Raw -LiteralPath `$Stub }
function Start-Process { param(`$FilePath, `$ArgumentList, `$WorkingDirectory) Write-Host "STUB no launch: `$FilePath" }
& `$Up -Roles dev1,dev2,infra
"@
$o = (& powershell.exe -NoProfile -File $upWrap -Stub $live -Up (Join-Path $swarm 'up.ps1') 2>&1 | Out-String)
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

Remove-Item -Recurse -Force $tmp
Remove-Item Env:SWARM_STATE_DIR
if ($fail) { Write-Host "$fail FAILED"; exit 1 } else { Write-Host 'ALL PASSED' }
