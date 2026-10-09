# Records a session id for a role in swarm/state/sessions.json (called from inside a launched tab).
param([Parameter(Mandatory)][string]$StateDir, [Parameter(Mandatory)][string]$Key, [Parameter(Mandatory)][string]$Name, [Parameter(Mandatory)][string]$SessionId)
. (Join-Path (Split-Path -Parent $PSScriptRoot) 'lib.ps1')
Save-Session $StateDir $Key $Name $SessionId
