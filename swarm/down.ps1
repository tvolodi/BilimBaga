<# Stops the swarm sessions (bb-* sessions; never the reused infra session unless -IncludeInfra). #>
param([switch]$IncludeInfra)
$s = claude agents --json | ConvertFrom-Json
foreach ($a in $s) {
  if ($a.name -like 'bb-*' -or ($IncludeInfra -and $a.name -like 'ai-dala-infra*')) {
    Write-Host "stopping $($a.name) pid $($a.pid)"
    Stop-Process -Id $a.pid -Force
  }
}
