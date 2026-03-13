param(
    [string]$TaskName = "ClipSyncClient",
    [string]$ExePath = "",
    [string]$Arguments = ""
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($ExePath)) {
    $ExePath = Join-Path $repoRoot "dist\clipsync-client.exe"
}
if ([string]::IsNullOrWhiteSpace($Arguments)) {
    $configPath = Join-Path $repoRoot "configs\client.json"
    $Arguments = "-config `"$configPath`""
}

if (!(Test-Path $ExePath)) {
    throw "Executable not found: $ExePath"
}

$action = New-ScheduledTaskAction -Execute $ExePath -Argument $Arguments
$trigger = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Description "ClipSync client autostart" -Force | Out-Null

Write-Host "Registered task: $TaskName"
Write-Host "Exe: $ExePath"
Write-Host "Args: $Arguments"
