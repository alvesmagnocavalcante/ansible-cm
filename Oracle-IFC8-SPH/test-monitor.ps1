#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$script:processPresent = $false
$script:logContent = ''
$script:readFails = $false
$script:logTime = Get-Date

# Mock only external reads; never query or modify the actual IFC8 installation.
function Get-CimInstance {
    param($ClassName, $Filter)
    if ($script:processPresent) {
        [pscustomobject]@{ CommandLine = 'IfcApplication.exe C:\Fidelio\Ifc8.Net\IfcApplication\SPH\Ifc8NetConfigSPH.Xml' }
    }
}
function Get-Item {
    param($LiteralPath)
    if ($script:readFails) { throw 'Log indisponivel' }
    [pscustomobject]@{ LastWriteTime = $script:logTime }
}
function Get-Content {
    param($LiteralPath, [switch]$Raw)
    $script:logContent
}

. "$PSScriptRoot\monitor-ifc8.ps1" -Once | Out-Null

function Assert-Status {
    param([string]$Expected)
    $actual = Get-IfcHealth
    if ($actual.Status -ne $Expected) {
        throw "Esperado $Expected, obtido $($actual.Status): $($actual.Reason)"
    }
}

Assert-Status 'OFFLINE'
$script:processPresent = $true
Assert-Status 'INDETERMINADO'
$script:logContent = @'
<MonItem ObjType="Ifc" Type="StateLink">Alive</MonItem>
<MonItem Type="StateComm" ObjType="Ifc">Sync</MonItem>
<MonItem ObjType='Pms' Type='StateLink'>Alive</MonItem>
<MonItem ObjType="Pms" Type="StateComm">Sync</MonItem>
'@
Assert-Status 'ONLINE'
$healthy = $script:logContent
$script:logContent += '<MonItem ObjType="Pms" Type="StateComm">Off</MonItem>'
Assert-Status 'OFFLINE'
$script:logContent += '<MonItem ObjType="Pms" Type="StateComm">Sync</MonItem>'
Assert-Status 'ONLINE'
$script:logContent += '<MonItem ObjType="Pms" Type="StateComm">'
Assert-Status 'ONLINE'
$script:readFails = $true
Assert-Status 'INDETERMINADO'
$script:readFails = $false
$script:logContent = $healthy
$script:logTime = (Get-Date).AddHours(-1)
Assert-Status 'ONLINE'
$MaxLogAgeSeconds = 180
Assert-Status 'INDETERMINADO'
Write-Host 'OK: 9 cenarios de processo, log, queda, recuperacao e idade do log.'
