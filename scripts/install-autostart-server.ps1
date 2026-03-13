param(
    [string]$TaskName = "ClipSyncServer",
    [string]$ExePath = "",
    [string]$Arguments = "",
    [switch]$AtStartup
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($ExePath)) {
    $ExePath = Join-Path $repoRoot "dist\clipsync-server.exe"
}
if ([string]::IsNullOrWhiteSpace($Arguments)) {
    $configPath = Join-Path $repoRoot "configs\server.json"
    $Arguments = "-config `"$configPath`""
}

if (!(Test-Path $ExePath)) {
    throw "Executable not found: $ExePath"
}

$action = New-ScheduledTaskAction -Execute $ExePath -Argument $Arguments
$trigger = if ($AtStartup) { New-ScheduledTaskTrigger -AtStartup } else { New-ScheduledTaskTrigger -AtLogOn }
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Description "ClipSync server autostart" -Force | Out-Null

Write-Host "Registered task: $TaskName"
Write-Host "Trigger: $($AtStartup ? 'AtStartup' : 'AtLogOn')"
Write-Host "Exe: $ExePath"
Write-Host "Args: $Arguments"
