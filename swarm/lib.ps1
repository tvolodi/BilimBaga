# Shared helpers for up.ps1 / ensure-up.ps1 / down.ps1 / install-ensure-task.ps1 (dot-source only).
# No decisions live here: roster parsing, token expansion, session-id state, launch command building.

$script:SwarmDir = $PSScriptRoot
$script:RepoRoot = Split-Path -Parent $PSScriptRoot
$script:TaskName = 'BilimBaga-Swarm-EnsureUp'

function Get-Roster {
  Get-Content -Raw -LiteralPath (Join-Path $script:SwarmDir 'roster.json') | ConvertFrom-Json
}

# Main checkout: roster.main_checkout (absolute path) or "auto" = parent of git's common dir (works from any worktree).
function Get-MainCheckout($Roster) {
  $m = $Roster.main_checkout
  if ($m -and $m -ne 'auto') { return ([IO.Path]::GetFullPath($m)) }
  try {
    $g = (& git -C $script:RepoRoot rev-parse --git-common-dir 2>$null)
    if ($LASTEXITCODE -eq 0 -and $g) {
      $g = "$g".Trim()
      $abs = if ([IO.Path]::IsPathRooted($g)) { [IO.Path]::GetFullPath($g) } else { [IO.Path]::GetFullPath((Join-Path $script:RepoRoot $g)) }
      return (Split-Path -Parent $abs)
    }
  } catch {}
  return $script:RepoRoot
}

# SWARM_STATE_DIR overrides the location (tests only).
function Get-StateDir($Roster) {
  if ($env:SWARM_STATE_DIR) { return $env:SWARM_STATE_DIR }
  Join-Path (Get-MainCheckout $Roster) ($Roster.state_dir -replace '/', '\')
}

function Expand-Tokens([string]$Text, $Roster) {
  if (-not $Text) { return $Text }
  $main = Get-MainCheckout $Roster
  $t = $Text.Replace('{preamble}', [string]$Roster.preamble)
  $t = $t.Replace('{main}', $main).Replace('{repo}', $script:RepoRoot)
  $t.Replace('{infra}', (Join-Path (Split-Path -Parent $main) 'ai-dala-infra'))
}

function Expand-Path([string]$Text, $Roster) { (Expand-Tokens $Text $Roster) -replace '/', '\' }

# Live sessions. -AgentsJsonFile is the test stub; otherwise `claude agents --json`. Throws if unreadable.
function Get-LiveAgents([string]$AgentsJsonFile) {
  if ($AgentsJsonFile) {
    $raw = Get-Content -Raw -LiteralPath $AgentsJsonFile
  } else {
    $raw = (& claude agents --json) -join "`n"
    if ($LASTEXITCODE -ne 0) { throw "claude agents --json failed (exit $LASTEXITCODE)" }
  }
  if (-not $raw -or -not $raw.Trim()) { return @() }
  @($raw | ConvertFrom-Json)
}

# Strict variant for ensure-up.ps1 (fail-safe): returns the live list only when it is a JSON array with at least
# one entry carrying a non-empty string `name`. Anything else (command failure, empty/non-JSON output, null, an
# object, an empty array, entries with other field names) THROWS, so the caller does nothing: an unrecognised
# schema must never be read as "every session is dead".
function Get-RecognisedLiveAgents([string]$AgentsJsonFile) {
  if ($AgentsJsonFile) {
    $raw = Get-Content -Raw -LiteralPath $AgentsJsonFile
  } else {
    $raw = (& claude agents --json) -join "`n"
    if ($LASTEXITCODE -ne 0) { throw "claude agents --json failed (exit $LASTEXITCODE)" }
  }
  if (-not $raw -or -not $raw.Trim()) { throw 'empty output from claude agents --json' }
  if (-not $raw.TrimStart().StartsWith('[')) { throw 'claude agents --json is not a JSON array (unrecognised schema)' }
  try { $parsed = @(,($raw | ConvertFrom-Json) | ForEach-Object { $_ }) } catch { throw "claude agents --json is not valid JSON: $_" }
  $named = @($parsed | Where-Object { $_ -and $_.name -is [string] -and $_.name.Trim() })
  if ($named.Count -eq 0) { throw 'claude agents --json has no entries with a recognisable name field (empty list or unrecognised schema)' }
  $named
}

function Get-AgentSessionId($Agent) {
  foreach ($p in 'sessionId', 'session_id') {
    $v = $Agent.$p
    if ($v -and "$v" -match '^[0-9a-fA-F-]{36}$') { return "$v" }
  }
  $null
}

function Find-LiveAgent($Live, [string]$Name) { $Live | Where-Object { $_.name -like "$Name*" } | Select-Object -First 1 }

# --- session-id state (swarm/state/sessions.json, git-ignored) ---
function Read-Sessions([string]$StateDir) {
  $f = Join-Path $StateDir 'sessions.json'
  $h = @{}
  if (Test-Path -LiteralPath $f) {
    try { (Get-Content -Raw -LiteralPath $f | ConvertFrom-Json).PSObject.Properties | ForEach-Object { $h[$_.Name] = $_.Value } } catch {}
  }
  $h
}

function Save-Session([string]$StateDir, [string]$Key, [string]$Name, [string]$SessionId) {
  if (-not (Test-Path -LiteralPath $StateDir)) { New-Item -ItemType Directory -Path $StateDir -Force | Out-Null }
  $h = Read-Sessions $StateDir
  $h[$Key] = [pscustomobject]@{ session_id = $SessionId; name = $Name; updated_utc = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ') }
  $f = Join-Path $StateDir 'sessions.json'
  $tmp = "$f.tmp"
  ($h | ConvertTo-Json -Depth 4) | Set-Content -LiteralPath $tmp -Encoding UTF8
  Move-Item -LiteralPath $tmp -Destination $f -Force   # atomic replace
}

function Get-SavedSessionId($Sessions, [string]$Key) {
  $e = $Sessions[$Key]
  if ($e -and $e.session_id -match '^[0-9a-fA-F-]{36}$') { return [string]$e.session_id }
  $null
}

# Env cleaning: a launcher started from inside a Claude session leaks its session env; a child would not register for messaging.
$script:CleanEnv = 'Get-ChildItem Env: | Where-Object { $_.Name -match ''^(CLAUDECODE|CLAUDE_PID|CLAUDE_CODE_(CHILD_SESSION|MESSAGING_.*|SESSION_.*|ENTRYPOINT))$'' } | ForEach-Object { Remove-Item ("Env:" + $_.Name) }; '

function ConvertTo-PsLiteral([string]$s) { "'" + ($s -replace "'", "''") + "'" }

# Returns @{ Command; Mode ('resume'|'fresh'); SessionId } for one roster role.
function New-LaunchSpec($Role, $Roster, [string]$StateDir, $Sessions, [switch]$Fresh) {
  $mode  = if ($Role.permission_mode) { $Role.permission_mode } else { $Roster.permission_mode }
  $model = if ($Role.model) { $Role.model } else { $Roster.model }
  $common = "-n $($Role.name) --permission-mode $mode --settings $(ConvertTo-PsLiteral (Expand-Path $Role.settings $Roster))"
  if ($model) { $common += " --model $model" }
  $saved = if ($Fresh) { $null } else { Get-SavedSessionId $Sessions $Role.key }
  $newId = [guid]::NewGuid().ToString()
  $rec = Join-Path $script:SwarmDir 'bin\record-session.ps1'
  $recCall = "& $(ConvertTo-PsLiteral $rec) -StateDir $(ConvertTo-PsLiteral $StateDir) -Key $($Role.key) -Name $($Role.name) -SessionId"
  $freshPrompt = ConvertTo-PsLiteral (Expand-Tokens $Role.prompt $Roster)
  $freshCmd = "$recCall $newId; claude --session-id $newId $common $freshPrompt"
  if ($saved) {
    $rp = if ($Role.resume_prompt) { $Role.resume_prompt } else { $Role.prompt }
    $resPrompt = ConvertTo-PsLiteral (Expand-Tokens $rp $Roster)
    # if the saved session can no longer be resumed (claude exits non-zero), fall back to a fresh session so the role is never lost
    # fall back only when resume died right away (<20 s): a later non-zero exit is a normal stop (down.ps1) and must not spawn a new session
    $cmd = $script:CleanEnv + "`$t0 = Get-Date; claude --resume $saved $common $resPrompt; if (`$LASTEXITCODE -ne 0 -and ((Get-Date) - `$t0).TotalSeconds -lt 20) { $freshCmd }"
    return @{ Command = $cmd; Mode = 'resume'; SessionId = $saved }
  }
  @{ Command = ($script:CleanEnv + $freshCmd); Mode = 'fresh'; SessionId = $newId }
}

# Opens the command in a Windows Terminal tab (window bilimbaga-swarm) or a separate window.
# Caller guards with ShouldProcess; the wt invocations are issued one tab at a time.
function Start-RoleTab($Role, $Spec, $Roster, [bool]$HaveWt) {
  $cwd = Expand-Path $Role.cwd $Roster
  $enc = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($Spec.Command))
  if ($HaveWt) {
    Start-Process wt.exe -ArgumentList @('-w', 'bilimbaga-swarm', 'new-tab', '--title', $Role.name, '-d', $cwd, 'powershell.exe', '-NoExit', '-EncodedCommand', $enc)
    Start-Sleep -Milliseconds 800
  } else {
    Start-Process powershell.exe -WorkingDirectory $cwd -ArgumentList @('-NoExit', '-EncodedCommand', $enc)
  }
}

# Refresh saved session ids from live agents that expose one (a resumed session may get a new id). Callers skip under -WhatIf.
function Update-SessionsFromLive($Roster, $Live, [string]$StateDir) {
  foreach ($r in $Roster.roles) {
    $a = Find-LiveAgent $Live $r.name
    if (-not $a) { continue }
    $id = Get-AgentSessionId $a
    if ($id) { Save-Session $StateDir $r.key $r.name $id }
  }
}
