<#
 Swarm launcher (no logic): opens each roster role as a LIVE interactive claude session in a
 Windows Terminal tab (fallback: separate windows). Roles come from swarm/roster.json.
 Session ids are recorded in swarm/state/sessions.json; a role with a recorded id is relaunched
 with --resume <id> (same -n name and per-role settings), otherwise started fresh with a new --session-id.
 Usage: powershell -File swarm\up.ps1 [-Roles dev1,dev2,ba,uat,infra,supervisor] [-ForceInfra] [-Fresh] [-WhatIf]
 -Fresh           ignore recorded session ids (start new conversations)
 -WhatIf          print what would be launched; launches nothing, writes nothing
 -AgentsJsonFile  test stub replacing `claude agents --json`
#>
[CmdletBinding(SupportsShouldProcess)]
param(
  [string[]]$Roles = @(),            # default: every roster role, in roster order (supervisor last)
  [switch]$ForceInfra,
  [switch]$Fresh,
  [string]$AgentsJsonFile
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'lib.ps1')
$roster   = Get-Roster
$stateDir = Get-StateDir $roster
$Roles = @($Roles | ForEach-Object { $_ -split ',' } | Where-Object { $_ })   # -File passes 'a,b' as one string
if (-not $Roles.Count) { $Roles = @($roster.roles | ForEach-Object { $_.key }) }

$live = @()
try { $live = Get-LiveAgents $AgentsJsonFile } catch { Write-Warning "cannot list live sessions: $_" }
$haveWt = [bool](Get-Command wt.exe -ErrorAction SilentlyContinue)
if (-not $WhatIfPreference) { Update-SessionsFromLive $roster $live $stateDir }
$sessions = Read-Sessions $stateDir

foreach ($k in $Roles) {
  $r = $roster.roles | Where-Object { $_.key -eq $k }
  if (-not $r) { Write-Warning "unknown role '$k' (not in roster.json)"; continue }
  if ($r.reuse_live_prefix -and -not $ForceInfra) {
    $reuse = Find-LiveAgent $live $r.reuse_live_prefix
    if ($reuse) { Write-Host "${k}: reusing running session ($($reuse.name))"; continue }
  }
  if (Find-LiveAgent $live $r.name) { Write-Host "$k already running, skipped"; continue }
  $spec = New-LaunchSpec $r $roster $stateDir $sessions -Fresh:$Fresh
  Write-Host "launching $k ($($r.name)) mode=$($spec.Mode) session=$($spec.SessionId)"
  if ($WhatIfPreference) { Write-Host "  cwd: $(Expand-Path $r.cwd $roster)"; Write-Host "  cmd: $($spec.Command)" }
  if ($PSCmdlet.ShouldProcess($r.name, "launch $($spec.Mode)")) {
    if ($spec.Mode -eq 'fresh') { Save-Session $stateDir $r.key $r.name $spec.SessionId }
    Start-RoleTab $r $spec $roster $haveWt
  }
}
