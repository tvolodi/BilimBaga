<#
 Registers a USER-LEVEL Scheduled Task that runs swarm\ensure-up.ps1 at logon and then every 5 minutes.
 The task runs as the current user, interactive (Windows Terminal tabs need the desktop), no elevation.
 It points at the MAIN checkout's swarm\ensure-up.ps1 so it survives worktree changes.
 THIS SCRIPT IS NOT RUN BY THE SWARM: the user runs it once. Preview with -WhatIf.
 Uninstall: swarm\down.ps1 (or -Uninstall here).
 Usage: powershell -File swarm\install-ensure-task.ps1 [-WhatIf] [-Uninstall] [-IntervalMinutes 5]
#>
[CmdletBinding(SupportsShouldProcess)]
param([switch]$Uninstall, [int]$IntervalMinutes = 5)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'lib.ps1')
$roster = Get-Roster
$script = Join-Path (Get-MainCheckout $roster) 'swarm\ensure-up.ps1'
$name = $script:TaskName

if ($Uninstall) {
  if (-not (Get-ScheduledTask -TaskName $name -ErrorAction SilentlyContinue)) { Write-Host 'task not installed'; return }
  if ($PSCmdlet.ShouldProcess($name, 'Unregister-ScheduledTask')) { Unregister-ScheduledTask -TaskName $name -Confirm:$false }
  return
}

if (-not (Test-Path -LiteralPath $script)) { Write-Warning "ensure-up.ps1 not found in main checkout yet: $script (merge the PR first)" }
$user = "$env:USERDOMAIN\$env:USERNAME"
$arg = "-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$script`""
$action  = New-ScheduledTaskAction -Execute 'powershell.exe' -Argument $arg
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $user
# repeat every N minutes indefinitely after logon
$trigger.Repetition = (New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes $IntervalMinutes)).Repetition
$principal = New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Limited
$settings  = New-ScheduledTaskSettingsSet -StartWhenAvailable -MultipleInstances IgnoreNew -ExecutionTimeLimit (New-TimeSpan -Minutes 4) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries

Write-Host "task:     $name"
Write-Host "user:     $user (interactive, limited)"
Write-Host "action:   powershell.exe $arg"
Write-Host "trigger:  at logon, repeat every $IntervalMinutes min"
if ($PSCmdlet.ShouldProcess($name, 'Register-ScheduledTask')) {
  Register-ScheduledTask -TaskName $name -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Description 'BilimBaga swarm liveness check (swarm/ensure-up.ps1)' | Out-Null
  Write-Host 'registered'
}
