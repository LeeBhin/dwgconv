# tvopen.ps1 <dwg>... : open drawings one at a time in DWG TrueView (Autodesk engine) and report
# every extra window/dialog the process shows (e.g. "drawing file is not valid", recovery prompts).
# Prints "OPENED <file>" when the drawing window appears, "NOT OPENED <file>" otherwise.
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Files)
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
$exe = if ($env:TV_EXE) { $env:TV_EXE } else { "C:\Program Files\Autodesk\DWG TrueView 2027 - English\dwgviewr.exe" }
$timeout = if ($env:TV_TIMEOUT) { [int]$env:TV_TIMEOUT } else { 90 }
$textCond = New-Object Windows.Automation.PropertyCondition([Windows.Automation.AutomationElement]::ControlTypeProperty, [Windows.Automation.ControlType]::Text)

foreach ($File in $Files) {
    $p = Start-Process $exe -ArgumentList "/nologo", "`"$File`"" -PassThru
    $name = [IO.Path]::GetFileName($File)
    $start = Get-Date
    $openedAt = $null
    $seen = @{}
    while (((Get-Date) - $start).TotalSeconds -lt $timeout) {
        Start-Sleep -Milliseconds 1000
        $cond = New-Object Windows.Automation.PropertyCondition([Windows.Automation.AutomationElement]::ProcessIdProperty, $p.Id)
        try { $wins = [Windows.Automation.AutomationElement]::RootElement.FindAll([Windows.Automation.TreeScope]::Children, $cond) } catch { continue }
        foreach ($w in $wins) {
            $t = $w.Current.Name
            if ($t -like "*$name*") { if (-not $openedAt) { $openedAt = Get-Date }; continue }
            if ($t -match '^Autodesk DWG TrueView 2027( - \[Start\])?$') { continue }  # main window while loading
            $msg = ($w.FindAll([Windows.Automation.TreeScope]::Descendants, $textCond) | ForEach-Object { $_.Current.Name } | Where-Object { $_ }) -join ' | '
            $key = "$t :: $msg"
            if (-not $seen.ContainsKey($key)) { $seen[$key] = $true; "  DIALOG $name : $key" }
        }
        if ($openedAt -and ((Get-Date) - $openedAt).TotalSeconds -gt 6) { break }
    }
    if ($openedAt) { "OPENED $name" } else { "NOT OPENED $name" }
    Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
    Get-Process dwgviewr -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 1500
}
