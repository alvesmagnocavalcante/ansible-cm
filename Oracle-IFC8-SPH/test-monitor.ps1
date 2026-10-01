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

$env:IFC8_TELEGRAM_TOKEN = 'token-ficticio-de-teste'
$env:IFC8_TELEGRAM_CHAT_ID = '-12345'
$script:sendCount = 0
$script:sendFails = $false
$script:apiRejects = $false
function Invoke-RestMethod {
    param($Method, $Uri, $ContentType, $Body, $TimeoutSec)
    $script:sendCount++
    if ($Method -ne 'Post' -or $TimeoutSec -ne 15) { throw 'Requisicao incorreta' }
    $payload = [Text.Encoding]::UTF8.GetString($Body) | ConvertFrom-Json
    if ($payload.chat_id -ne '-12345' -or -not $payload.text) { throw 'Payload incorreto' }
    if ($script:sendFails) { throw "Erro HTTP com token: $Uri" }
    [pscustomobject]@{ ok = -not $script:apiRejects }
}

$state = @{ Status = ''; LastSuccess = [datetime]::MinValue; LastAttempt = [datetime]::MinValue }
$now = Get-Date
$online = [pscustomobject]@{ Status = 'ONLINE'; Reason = 'Alive/Sync' }
$offline = [pscustomobject]@{ Status = 'OFFLINE'; Reason = 'pms.comm=Off' }
Invoke-IfcTelegramNotification $online $now $state
if ($script:sendCount -ne 0) { throw 'Nao deve enviar ONLINE inicial' }
Invoke-IfcTelegramNotification $offline $now $state
Invoke-IfcTelegramNotification $offline $now.AddSeconds(30) $state
if ($script:sendCount -ne 1 -or $state.Status -ne 'OFFLINE') { throw 'Alerta inicial/duplicado incorreto' }
Invoke-IfcTelegramNotification $offline $now.AddSeconds(300) $state
if ($script:sendCount -ne 2) { throw 'Lembrete nao enviado' }
Invoke-IfcTelegramNotification $online $now.AddSeconds(330) $state
if ($script:sendCount -ne 3 -or $state.Status -ne 'ONLINE') { throw 'Recuperacao nao enviada' }
$script:sendFails = $true
$warnings = @(Invoke-IfcTelegramNotification $offline $now.AddSeconds(360) $state 3>&1)
if (($warnings | Out-String).Contains($env:IFC8_TELEGRAM_TOKEN)) { throw 'Token exposto no aviso de erro' }
if ($state.Status -ne 'ONLINE') { throw 'Envio com falha marcado como sucesso' }
$script:sendFails = $false
Invoke-IfcTelegramNotification $offline $now.AddSeconds(375) $state
if ($script:sendCount -ne 4) { throw 'Retry antecipado' }
Invoke-IfcTelegramNotification $offline $now.AddSeconds(390) $state
if ($script:sendCount -ne 5 -or $state.Status -ne 'OFFLINE') { throw 'Retry nao executado' }
$script:apiRejects = $true
$accepted = Send-IfcTelegram 'Teste recusado pela API' -WarningAction SilentlyContinue
if ($accepted) { throw 'Resposta ok=false aceita' }
Write-Host 'OK: Telegram com API simulada: payload, repeticao, recuperacao, retry e token protegido.'
