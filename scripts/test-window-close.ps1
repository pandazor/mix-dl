param([string]$Binary = (Join-Path $PSScriptRoot '..\bin\dl-manager.exe'))
$ErrorActionPreference = 'Stop'
$testData = Join-Path ([IO.Path]::GetTempPath()) ('dlm-close-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $testData | Out-Null
Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class DlmCloseProbe {
 public delegate bool EnumProc(IntPtr hwnd,IntPtr lparam);
 [DllImport("user32.dll")] static extern bool EnumWindows(EnumProc cb,IntPtr p);
 [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h,out uint pid);
 [DllImport("user32.dll",CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h,StringBuilder title,int count);
 [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h,uint m,IntPtr w,IntPtr l);
 public static IntPtr Find(uint pid) { IntPtr found=IntPtr.Zero; EnumWindows((h,p)=>{uint owner;GetWindowThreadProcessId(h,out owner);var title=new StringBuilder(512);GetWindowText(h,title,512);if(owner==pid && title.ToString()=="dl-manager"){found=h;return false;}return true;},IntPtr.Zero);return found; }
}
'@
$portProbe = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0)
$portProbe.Start()
$debugPort = $portProbe.LocalEndpoint.Port
$portProbe.Stop()
$savedBrowserArgs = $env:WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS
$env:WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS = '--remote-debugging-port=' + $debugPort
$process = Start-Process -FilePath (Resolve-Path -LiteralPath $Binary).Path -ArgumentList @('--data', ('"' + $testData + '"')) -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $testData "stdout.log") -RedirectStandardError (Join-Path $testData "stderr.log")
$env:WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS = $savedBrowserArgs
# Retain the native process handle so ExitCode stays available after exit.
$null = $process.Handle
try {
    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    do {
        Start-Sleep -Milliseconds 100
        $process.Refresh()
        $window = [DlmCloseProbe]::Find($process.Id)
        if ($process.HasExited) { throw 'Test application exited before its window was ready' }
    } while ($window -eq [IntPtr]::Zero -and [DateTime]::UtcNow -lt $deadline)
    if ($window -eq [IntPtr]::Zero) { throw 'Test application window not found' }
    # Wait for WebView2 navigation, not merely the native shell window.
    # Closing during controller creation would abort startup rather than exercise close.
    $ready = $false
    do {
        try {
            $targets = Invoke-RestMethod -Uri ("http://127.0.0.1:" + $debugPort + "/json") -TimeoutSec 1
            $ready = @($targets | Where-Object { $_.type -eq 'page' -and $_.url -match '^https?://' }).Count -gt 0
        } catch { }
        if (-not $ready) { Start-Sleep -Milliseconds 100 }
    } while (-not $ready -and [DateTime]::UtcNow -lt $deadline)
    if (-not $ready) { throw ('WebView2 did not finish initial navigation: ' + ($targets | ConvertTo-Json -Compress) + ' ' + (Get-Content -LiteralPath (Join-Path $testData 'stderr.log') -Raw)) }
    if (-not [DlmCloseProbe]::PostMessage($window, 0x10, [IntPtr]::Zero, [IntPtr]::Zero)) { throw 'Native close request failed' }
    if (-not $process.WaitForExit(8000)) { throw 'Native close left the application running in the background' }
    if ($process.ExitCode -ne 0) { throw ('Application exit code: ' + $process.ExitCode + ' ' + (Get-Content -LiteralPath (Join-Path $testData 'stderr.log') -Raw)) }
    Write-Output 'PASS: native close ended the isolated application process'
} finally {
    $process.Refresh()
    if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force }
    # Keep the isolated data directory for inspection; never touch user data.
}
