#requires -Version 5.1
[CmdletBinding()]
param(
    [string]$ConfigPath = 'C:\Fidelio\Ifc8.Net\IfcApplication\SPH\Ifc8NetConfigSPH.Xml',
    [string]$LogPath = 'C:\Fidelio\Ifc8.Net\IfcApplication\SPH\M87POS_SPH_Log.evt',
    [ValidateRange(5, 3600)][int]$IntervalSeconds = 30,
    [ValidateRange(30, 86400)][int]$RepeatAlertSeconds = 300,
    [ValidateRange(0, 86400)][int]$MaxLogAgeSeconds = 0,
    [switch]$Once,
    [switch]$NoPopup
)

$ErrorActionPreference = 'Stop'

function Get-IfcLogStates {
    param([string]$Content)
    $states = @{}
    # Read complete MonItem fragments; attribute order is not significant.
    foreach ($item in [regex]::Matches($Content, '(?is)<MonItem\b([^>]*)>([^<]*)</MonItem\s*>')) {
        $attributes = $item.Groups[1].Value
        $objectMatch = [regex]::Match($attributes, '(?i)\bObjType\s*=\s*["''](Ifc|Pms)["'']')
        $typeMatch = [regex]::Match($attributes, '(?i)\bType\s*=\s*["'']State(Link|Comm)["'']')
        if ($objectMatch.Success -and $typeMatch.Success) {
            $key = '{0}.{1}' -f $objectMatch.Groups[1].Value.ToLowerInvariant(), $typeMatch.Groups[1].Value.ToLowerInvariant()
            $states[$key] = $item.Groups[2].Value.Trim()
        }
    }
    return $states
}

function Get-IfcHealth {
    try {
        $running = Get-CimInstance Win32_Process -Filter "Name='IfcApplication.exe'" |
            Where-Object { $_.CommandLine -and $_.CommandLine.IndexOf($ConfigPath, [StringComparison]::OrdinalIgnoreCase) -ge 0 } |
            Select-Object -First 1
        if (-not $running) {
            return [pscustomobject]@{ Status = 'OFFLINE'; Reason = 'Processo IFC8 SPH ausente.' }
        }
        $file = Get-Item -LiteralPath $LogPath
        if ($MaxLogAgeSeconds -gt 0 -and ((Get-Date) - $file.LastWriteTime).TotalSeconds -gt $MaxLogAgeSeconds) {
            return [pscustomobject]@{ Status = 'INDETERMINADO'; Reason = 'Log sem atualizacao dentro do prazo configurado.' }
        }
        $states = Get-IfcLogStates (Get-Content -LiteralPath $LogPath -Raw)
        $expected = @{ 'ifc.link' = 'Alive'; 'ifc.comm' = 'Sync'; 'pms.link' = 'Alive'; 'pms.comm' = 'Sync' }
        $missing = @()
        $offline = @()
        foreach ($key in ($expected.Keys | Sort-Object)) {
            if (-not $states.ContainsKey($key)) {
                $missing += $key
            } elseif ($states[$key] -ne $expected[$key]) {
                $offline += "$key=$($states[$key])"
            }
        }
        if ($offline.Count -gt 0) {
            return [pscustomobject]@{ Status = 'OFFLINE'; Reason = $offline -join '; ' }
        }
        if ($missing.Count -gt 0) {
            return [pscustomobject]@{ Status = 'INDETERMINADO'; Reason = 'Estados ausentes no log: ' + ($missing -join ', ') }
        }
        return [pscustomobject]@{ Status = 'ONLINE'; Reason = 'IFC e PMS: Alive/Sync (ultimos estados registrados).' }
    } catch {
        return [pscustomobject]@{ Status = 'INDETERMINADO'; Reason = $_.Exception.Message }
    }
}

if ($Once) {
    Get-IfcHealth
    return
}

$notification = $null
try {
    if (-not $NoPopup) {
        Add-Type -AssemblyName System.Windows.Forms
        Add-Type -AssemblyName System.Drawing
        $notification = New-Object System.Windows.Forms.NotifyIcon
        $notification.Icon = [System.Drawing.SystemIcons]::Warning
        $notification.Text = 'Monitor IFC8 SPH'
        $notification.Visible = $true
    }
    $lastStatus = ''
    $lastAlert = [datetime]::MinValue
    while ($true) {
        $health = Get-IfcHealth
        $now = Get-Date
        $changed = $health.Status -ne $lastStatus
        if ($changed) {
            Write-Host ('{0:yyyy-MM-dd HH:mm:ss} [{1}] {2}' -f $now, $health.Status, $health.Reason)
        }
        if ($health.Status -ne 'ONLINE' -and ($changed -or ($now - $lastAlert).TotalSeconds -ge $RepeatAlertSeconds)) {
            if (-not $changed) {
                Write-Host ('{0:yyyy-MM-dd HH:mm:ss} [{1}] {2}' -f $now, $health.Status, $health.Reason)
            }
            if ($notification) {
                $notification.ShowBalloonTip(10000, "IFC8 SPH: $($health.Status)", $health.Reason, [System.Windows.Forms.ToolTipIcon]::Warning)
                [System.Media.SystemSounds]::Exclamation.Play()
            }
            $lastAlert = $now
        }
        $lastStatus = $health.Status
        Start-Sleep -Seconds $IntervalSeconds
    }
} finally {
    if ($notification) {
        $notification.Visible = $false
        $notification.Dispose()
    }
}
