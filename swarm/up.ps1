<#
 Swarm launcher (no logic): opens each role as a LIVE interactive claude session in a
 Windows Terminal tab (fallback: separate windows). See swarm/README.md.
 Usage: powershell -File swarm\up.ps1 [-Roles dev1,dev2,ba,uat,infra,supervisor] [-ForceInfra]
#>
param(
  [string[]]$Roles = @('dev1','dev2','ba','uat','infra','supervisor'),  # supervisor last
  [switch]$ForceInfra
)
$ErrorActionPreference = 'Stop'
$Repo  = Split-Path -Parent $PSScriptRoot
$Infra = Join-Path (Split-Path -Parent $Repo) 'ai-dala-infra'
$Mode  = 'bypassPermissions'   # one common permission mode, see PROTOCOL.md section 2

$live = @()
try { $live = (claude agents --json | ConvertFrom-Json) | ForEach-Object { $_.name } } catch {}

$def = @{
  dev1       = @{ name='bb-dev1';       cwd="$Repo\.claude\worktrees\dev1"; prompt='Startup: read swarm/roles/dev1.md (main checkout path C:\Users\tvolo\dev\ai-dala\BilimBaga\swarm\roles\dev1.md) plus _common.md and PROTOCOL.md, do the startup handshake, then start your /loop tick.' }
  dev2       = @{ name='bb-dev2';       cwd="$Repo\.claude\worktrees\dev2"; prompt='Startup: read swarm/roles/dev2.md (main checkout path C:\Users\tvolo\dev\ai-dala\BilimBaga\swarm\roles\dev2.md) plus _common.md and PROTOCOL.md, do the startup handshake, then start your /loop tick.' }
  ba         = @{ name='bb-ba';         cwd=$Repo; prompt='Startup: read swarm/roles/ba.md plus _common.md and PROTOCOL.md, do the startup handshake, then start your /loop tick.' }
  uat        = @{ name='bb-uat';        cwd=$Repo; prompt='Startup: read swarm/roles/uat.md plus _common.md and PROTOCOL.md, do the startup handshake, then start your /loop tick.' }
  infra      = @{ name='bb-infra';      cwd=$Infra; prompt="Startup: read $Repo\swarm\roles\infra.md plus _common.md and PROTOCOL.md in that repo, do the startup handshake, then start your /loop tick." }
  supervisor = @{ name='bb-supervisor'; cwd=$Repo; prompt='/loop Startup: read swarm/roles/supervisor.md plus _common.md, PROTOCOL.md, RETRO.md, then run the Supervisor Tick forever (dynamic cadence).' }
}
$settings = @{ dev1='dev1'; dev2='dev2'; ba='ba'; uat='uat'; infra='infra'; supervisor='supervisor' }

$haveWt = [bool](Get-Command wt.exe -ErrorAction SilentlyContinue)
$wtArgs = @()
foreach ($r in $Roles) {
  $d = $def[$r]
  if ($r -eq 'infra' -and -not $ForceInfra -and ($live | Where-Object { $_ -like 'ai-dala-infra*' })) { Write-Host "infra: reusing running session ($($live | Where-Object { $_ -like 'ai-dala-infra*' }))"; continue }
  if ($live | Where-Object { $_ -like "$($d.name)*" }) { Write-Host "$r already running, skipped"; continue }
  $sf  = Join-Path $PSScriptRoot "roles\$($settings[$r]).settings.json"
  # a launcher started from inside a Claude session leaks its session env; a child would not register for messaging
  $clean = "Get-ChildItem Env: | Where-Object { `$_.Name -match '^(CLAUDECODE|CLAUDE_PID|CLAUDE_CODE_(CHILD_SESSION|MESSAGING_.*|SESSION_.*|ENTRYPOINT))`$' } | ForEach-Object { Remove-Item (`"Env:`" + `$_.Name) }; "
  $cmd = $clean + "claude -n $($d.name) --permission-mode $Mode $(if (Test-Path $sf) { "--settings '$sf'" }) '$($d.prompt -replace "'","''")'"
  $enc = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($cmd))
  if ($haveWt) {
    if ($wtArgs.Count) { $wtArgs += ';' }
    $wtArgs += @('-w','bilimbaga-swarm','new-tab','--title',$d.name,'-d',$d.cwd,'powershell.exe','-NoExit','-EncodedCommand',$enc)
  } else {
    Start-Process powershell.exe -WorkingDirectory $d.cwd -ArgumentList @('-NoExit','-EncodedCommand',$enc)
  }
  Write-Host "launching $r ($($d.name))"
}
if ($haveWt -and $wtArgs.Count) { Start-Process wt.exe -ArgumentList $wtArgs }
