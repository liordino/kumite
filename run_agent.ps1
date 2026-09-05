# Autonomous overnight runner (run_agent.ps1) — v2, print-mode design
#
# Why v2: the v1 script ran `pi -r` interactively. When a session or usage
# limit cut the agent's turn, the CLI stayed open waiting for input, so this
# script blocked on the same invocation forever — the overnight stall of
# 2026-09-05 (00:08 -> 07:38) that logged nothing, because the failure never
# surfaced as a process exit. v2 drives each turn in print mode
# (`pi --session <file> -p "<instruction>"`): one bounded agent turn per
# loop iteration. A limit cutoff ends the turn and pi exits, so the stall
# cannot recur and every turn is observable (exit code + duration).
#
# The instruction doubles as the resume prompt. It forces the agent to
# re-establish ground truth (git + tests) before trusting anything it
# remembers from before the cutoff, restates the autonomy contract, and
# defines the completion signal: the agent creates $DoneSentinel when the
# brief's V1 scope is complete and verified — the script's only exit 0.
# (If the agent finishes but gets cut before writing the sentinel, the next
# resume re-verifies and creates it — self-correcting.)
#
# Known limits: no per-turn timeout (pi's own HTTP timeouts bound a turn;
# a genuinely hung turn needs a manual Ctrl-C). Retries are unlimited by
# default ($MaxRetries = 0) because usage windows reset on their own.

# Configuration
$LogFile = "agent_errors.log"
$TempErrFile = ".agent_tmp_err.log"
$WaitTimeSeconds = 900    # 15 minutes - wait after a no-progress turn
$QuickExitSeconds = 120   # turns shorter than this made no real progress
$MaxRetries = 0           # 0 = unlimited retries
$DoneSentinel = ".agent_done"
$SessionsRoot = Join-Path $env:USERPROFILE ".pi\agent\sessions"

$ResumeInstruction = "AUTONOMOUS RESUME (unattended continuation via run_agent.ps1): the previous agent turn ended - either normally or cut off by a session or usage limit. The user is not present. Re-establish ground truth before continuing: run 'git log --oneline -8' and 'git status --short', run 'go test ./...' and 'cd client && npm run check', and read the latest commit message to identify the last completed milestone. Then continue the remaining V1 build order per KUMITE-BRIEF.md, AGENTS.md and CONTEXT.md. Work autonomously: no questions, commit after each verified milestone, note deviations in the commit message. If the entire V1 scope is already complete and verified (gofmt, go vet, go test, svelte-check, both builds green), create the file .agent_done and stop."

Push-Location $PSScriptRoot
$ErrorActionPreference = "Continue"
$RetryCount = 0

# Each launch starts a fresh run: a leftover sentinel from a previous run
# must not end this one.
if (Test-Path $DoneSentinel) { Remove-Item $DoneSentinel -ErrorAction SilentlyContinue }

$Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
Write-Output "[$Timestamp] Starting autonomous loop (print mode: one bounded agent turn per iteration)..." | Add-Content $LogFile

while ($true) {
    # Resolve the newest recorded session for this repo BEFORE each turn.
    # The first turn of a fresh checkout starts a new session (no session
    # flags); every later turn pins the file explicitly - bare `pi -c`
    # continues the most recent session globally, which could belong to
    # another project.
    $sessionArg = @()
    $latestSession = Get-ChildItem -Path $SessionsRoot -Recurse -Filter *.jsonl -ErrorAction SilentlyContinue |
        Where-Object { $_.DirectoryName -like "*kumite*" } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
    if ($latestSession) {
        $sessionArg = @("--session", $latestSession.FullName)
        $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        Write-Output "[$Timestamp] Pinning session: $($latestSession.Name)" | Add-Content $LogFile
    }

    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    & pi @sessionArg -p $ResumeInstruction 2> $TempErrFile
    $sw.Stop()

    # Completion: the agent creates the sentinel when the V1 scope is done.
    if (Test-Path $DoneSentinel) {
        $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        Write-Output "[$Timestamp] Completion sentinel found - V1 scope complete. Exiting." | Add-Content $LogFile
        if (Test-Path $TempErrFile) { Remove-Item $TempErrFile -ErrorAction SilentlyContinue }
        Pop-Location
        exit 0
    }

    $madeProgress = ($LASTEXITCODE -eq 0) -and ($sw.Elapsed.TotalSeconds -ge $QuickExitSeconds)

    if ($madeProgress) {
        $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        Write-Output ("[$Timestamp] Turn completed in {0:n0}s." -f $sw.Elapsed.TotalSeconds) | Add-Content $LogFile
        if (Test-Path $TempErrFile) { Remove-Item $TempErrFile -ErrorAction SilentlyContinue }
        Start-Sleep -Seconds 30
        continue
    }

    # No progress: quick exit (limit not reset yet), nonzero exit, or crash.
    $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "----------------------------------------" | Add-Content $LogFile
    "[$Timestamp] NO PROGRESS: pi exited $LASTEXITCODE after $([int]$sw.Elapsed.TotalSeconds)s (no-progress retry $RetryCount of $(if ($MaxRetries -gt 0) { $MaxRetries } else { 'unlimited' }))" | Add-Content $LogFile
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

    Write-Host "No progress detected. Beginning countdown window..."

    # Live downtime tracking with remaining-time display (v1 UX kept).
    $StartPause = [DateTimeOffset]::Now.ToUnixTimeSeconds()
    while ($true) {
        $CurrentTime = [DateTimeOffset]::Now.ToUnixTimeSeconds()
        $Elapsed = $CurrentTime - $StartPause
        $Remaining = $WaitTimeSeconds - $Elapsed
        if ($Remaining -lt 0) { $Remaining = 0 }

        $Hours = [math]::Floor($Elapsed / 3600)
        $Minutes = [math]::Floor(($Elapsed % 3600) / 60)
        $RemMinutes = [math]::Ceiling($Remaining / 60)

        [Console]::Write("`r[Downtime Status] Elapsed: {0:D2} hours {1:D2} minutes | ~{2} min until retry (no-progress retry #{3}).   " -f $Hours, $Minutes, $RemMinutes, $RetryCount)

        Start-Sleep -Seconds 60

        if ($Elapsed -ge $WaitTimeSeconds) {
            Write-Host "" # Fresh line
            Write-Host "Window reached. Resuming agent execution..."
            break
        }
    }
}