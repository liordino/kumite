# Autonomous overnight runner (run_agent.ps1) — improvements
#
# Changes over the initial version:
#   - Runs from the script's own directory (Push-Location $PSScriptRoot), so it
#     works no matter where it is launched from.
#   - $MaxRetries cap (0 = unlimited, the default preserves prior behavior).
#   - Retry counter + session-end summary logged.
#   - Countdown shows remaining time, not only elapsed.
#   - Temp error file cleanup is failure-tolerant (Remove-Item -ErrorAction SilentlyContinue).

# Configuration
$LogFile = "agent_errors.log"
$WaitTimeSeconds = 900   # 15 minutes
$TempErrFile = ".agent_tmp_err.log"
$MaxRetries = 0          # 0 = unlimited retries (usage limits reset on their own schedule)

Push-Location $PSScriptRoot

$ErrorActionPreference = "Continue"
$RetryCount = 0
$Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
Write-Output "[$Timestamp] Starting autonomous overnight loop using 'pi -r'..." | Add-Content $LogFile

while ($true) {
    & pi -r 2> $TempErrFile

    if ($LASTEXITCODE -eq 0) {
        $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        Write-Output "[$Timestamp] Repository agent loop completed successfully! (retries used: $RetryCount)" | Add-Content $LogFile
        if (Test-Path $TempErrFile) { Remove-Item $TempErrFile -ErrorAction SilentlyContinue }
        Pop-Location
        break
    }

    # Log the crash details
    $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "----------------------------------------" | Add-Content $LogFile
    "[$Timestamp] ERROR: 'pi -r' exited with status $LASTEXITCODE (retry $RetryCount" +
        $(if ($MaxRetries -gt 0) { "/$MaxRetries" } else { "/unlimited" }) + ")" | Add-Content $LogFile

    if ((Test-Path $TempErrFile) -and (Get-Item $TempErrFile).Length -gt 0) {
        Get-Content $TempErrFile | Add-Content $LogFile
    } else {
        "No detailed error output captured." | Add-Content $LogFile
    }
    "----------------------------------------" | Add-Content $LogFile
    if (Test-Path $TempErrFile) { Remove-Item $TempErrFile -ErrorAction SilentlyContinue }

    if ($MaxRetries -gt 0 -and $RetryCount -ge $MaxRetries) {
        $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        "[$Timestamp] Max retries ($MaxRetries) reached. Giving up." | Add-Content $LogFile
        Write-Host "Max retries reached. Giving up."
        Pop-Location
        exit 1
    }
    $RetryCount++

    Write-Host "Agent rate-limited or halted. Beginning countdown window..."

    # Live downtime tracking with remaining-time display
    $StartPause = [DateTimeOffset]::Now.ToUnixTimeSeconds()
    while ($true) {
        $CurrentTime = [DateTimeOffset]::Now.ToUnixTimeSeconds()
        $Elapsed = $CurrentTime - $StartPause
        $Remaining = $WaitTimeSeconds - $Elapsed
        if ($Remaining -lt 0) { $Remaining = 0 }

        $Hours = [math]::Floor($Elapsed / 3600)
        $Minutes = [math]::Floor(($Elapsed % 3600) / 60)
        $RemMinutes = [math]::Ceiling($Remaining / 60)

        $HoursStr = $Hours.ToString("D2")
        $MinutesStr = $Minutes.ToString("D2")

        [Console]::Write("`r[Downtime Status] Elapsed: $HoursStr hours $MinutesStr minutes | ~$RemMinutes min until retry (retry #$RetryCount).   ")

        Start-Sleep -Seconds 60

        if ($Elapsed -ge $WaitTimeSeconds) {
            Write-Host "" # Fresh line
            Write-Host "15-minute window reached. Attempting to resume agent execution now..."
            break
        }
    }
}