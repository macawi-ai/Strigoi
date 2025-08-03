# Windows demonstration of parent-child bypass
# Shows how credentials are exposed when launching MCP servers

Write-Host "=== Parent-Child Bypass Demo (Windows) ===" -ForegroundColor Yellow
Write-Host ""

# Windows doesn't have YAMA, but has similar parent-child relationships
Write-Host "[*] Windows uses different security model than Linux YAMA" -ForegroundColor Cyan
Write-Host "[*] But parent processes can still access child process data" -ForegroundColor Cyan
Write-Host ""

# Method 1: Using Process Monitor or API Monitor (if available)
Write-Host "=== Method 1: Process Creation Monitoring ===" -ForegroundColor Green
Write-Host "[*] Launching server with credentials in command line..."
Write-Host "[*] Command: python echo-server.py 'user:SuperSecret123@db.internal:5432/production'"
Write-Host ""

# Start the process
$proc = Start-Process python -ArgumentList "echo-server.py", "user:SuperSecret123@db.internal:5432/production" -PassThru -WindowStyle Hidden

Start-Sleep -Seconds 1

# Method 2: WMI Process Inspection
Write-Host "=== Method 2: WMI Process Inspection ===" -ForegroundColor Green
Write-Host "[*] Using WMI to inspect process command line..."

$processes = Get-WmiObject Win32_Process | Where-Object { $_.CommandLine -like "*echo-server*" -and $_.CommandLine -like "*Secret*" }

foreach ($p in $processes) {
    Write-Host "[!] Found process with exposed credentials:" -ForegroundColor Red
    Write-Host "    PID: $($p.ProcessId)"
    Write-Host "    Command: $($p.CommandLine)" -ForegroundColor Red
}

# Method 3: Get-Process and StartInfo
Write-Host ""
Write-Host "=== Method 3: PowerShell Process Inspection ===" -ForegroundColor Green
Write-Host "[*] Using Get-Process to find the server..."

$serverProc = Get-Process python -ErrorAction SilentlyContinue | Where-Object { $_.Id -eq $proc.Id }
if ($serverProc) {
    Write-Host "[*] Found server process: PID $($serverProc.Id)"
    
    # Note: StartInfo.Arguments often empty for security, but WMI still shows it
    Write-Host "[*] Process info available to parent/same-user processes"
}

# Cleanup
Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "=== Windows-Specific Risks ===" -ForegroundColor Yellow
Write-Host "[!] On Windows, credentials are exposed through:" -ForegroundColor Red
Write-Host "    - WMI queries (available to same user)"
Write-Host "    - Process creation APIs"
Write-Host "    - Event logs (if process creation auditing enabled)"
Write-Host "    - Debugging APIs (for parent processes)"
Write-Host "    - Handle inheritance (child processes)"
Write-Host ""

# Method 4: Demonstrate handle inheritance issue
Write-Host "=== Method 4: Handle Inheritance ===" -ForegroundColor Green
Write-Host "[*] Windows specific: Child processes inherit handles from parents"
Write-Host "[*] This includes:"
Write-Host "    - File handles (including credential files)"
Write-Host "    - Registry keys"
Write-Host "    - Named pipes"
Write-Host "    - Synchronization objects"
Write-Host ""

Write-Host "=== Summary ===" -ForegroundColor Yellow
Write-Host "[!] Windows lacks YAMA but has similar vulnerabilities:" -ForegroundColor Red
Write-Host "    - Command line arguments visible via WMI"
Write-Host "    - Parent processes can debug children"
Write-Host "    - Handle inheritance leaks access"
Write-Host "    - No process isolation within same user"
Write-Host ""
Write-Host "[*] This affects MCP servers on Windows:" -ForegroundColor Cyan
Write-Host "    - Database credentials in command line"
Write-Host "    - API keys visible to any same-user process"
Write-Host "    - Parent process (Claude) can access all child data"
Write-Host "    - No effective isolation mechanism"
Write-Host ""
Write-Host "[*] Mitigation: Use Windows Credential Manager or secure IPC!" -ForegroundColor Green