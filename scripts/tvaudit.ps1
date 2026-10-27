# tvaudit.ps1 <dwg>...: open each drawing in DWG TrueView, accept recovery, and print the error lines of
# AutoCAD's recovery/audit report (read from the command history via UI Automation).
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Files)
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
$exe = "C:\Program Files\Autodesk\DWG TrueView 2027 - English\dwgviewr.exe"
$AE = [Windows.Automation.AutomationElement]
foreach ($File in $Files) {
    "== $([IO.Path]::GetFileName($File))"
    $p = Start-Process $exe -ArgumentList "/nologo", "`"$File`"" -PassThru
    $ws = New-Object -ComObject WScript.Shell
    for ($k = 0; $k -lt 2; $k++) {
        for ($i = 0; $i -lt 30; $i++) { Start-Sleep 2; if ($ws.AppActivate("Open Drawing - Damaged File")) { break } }
        Start-Sleep 1; $ws.SendKeys("{ENTER}")
    }
    Start-Sleep 3
    $null = $ws.AppActivate("Autodesk DWG TrueView 2027"); Start-Sleep 1; $ws.SendKeys("{F2}"); Start-Sleep 3
    $all = $AE::RootElement.FindAll([Windows.Automation.TreeScope]::Descendants, (New-Object Windows.Automation.PropertyCondition($AE::ProcessIdProperty, $p.Id)))
    $text = ""
    foreach ($e in $all) {
        try { $t = $e.GetCurrentPattern([Windows.Automation.TextPattern]::Pattern).DocumentRange.GetText(200000); if ($t.Length -gt $text.Length) { $text = $t } } catch {}
    }
    ($text -split "`r?`n") | Where-Object { $_ -match 'Error|Missing|Invalid|invalid|Erased|fixed|discarded|Incorrect|not |bad|Reading handle' } | Select-Object -Unique
    Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
    Start-Sleep 2
}
