<#
 Stops the swarm sessions (bb-* sessions; never the reused infra session unless -IncludeInfra) and
 UNINSTALLS the ensure-up Scheduled Task first (otherwise it would relaunch what is being stopped).
 Usage: powershell -File swarm\down.ps1 [-IncludeInfra] [-KeepTask] [-WhatIf] [-AgentsJsonFile stub.json]
#>
[CmdletBinding(SupportsShouldProcess)]
param([switch]$IncludeInfra, [switch]$KeepTask, [string]$AgentsJsonFile)
. (Join-Path $PSScriptRoot 'lib.ps1')

if (-not $KeepTask) {
  $t = Get-ScheduledTask -TaskName $script:TaskName -ErrorAction SilentlyContinue
  if ($t) {
    if ($PSCmdlet.ShouldProcess($script:TaskName, 'Unregister-ScheduledTask')) {
      Unregister-ScheduledTask -TaskName $script:TaskName -Confirm:$false
      Write-Host "uninstalled scheduled task $($script:TaskName)"
    }
  } else { Write-Host "scheduled task $($script:TaskName) not installed" }
}

$s = Get-LiveAgents $AgentsJsonFile
foreach ($a in $s) {
  if ($a.name -like 'bb-*' -or ($IncludeInfra -and $a.name -like 'ai-dala-infra*')) {
    if ($PSCmdlet.ShouldProcess("$($a.name) pid $($a.pid)", 'Stop-Process')) {
      Write-Host "stopping $($a.name) pid $($a.pid)"
      Stop-Process -Id $a.pid -Force
    }
  }
}
