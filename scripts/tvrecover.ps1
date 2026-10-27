# tvrecover.ps1 <dwg>: open a drawing in DWG TrueView, press "Recover" when the damaged-file dialog
# appears, and print the text AutoCAD shows afterwards (the recovery/audit report).
param([string]$File, [int]$Seconds = 120)
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
$exe = "C:\Program Files\Autodesk\DWG TrueView 2027 - English\dwgviewr.exe"
$p = Start-Process $exe -ArgumentList "/nologo", "`"$File`"" -PassThru
$AE = [Windows.Automation.AutomationElement]
$pidCond = New-Object Windows.Automation.PropertyCondition($AE::ProcessIdProperty, $p.Id)
$start = Get-Date
$pressed = $false
$lines = [ordered]@{}
while (((Get-Date) - $start).TotalSeconds -lt $Seconds) {
    Start-Sleep -Milliseconds 1500
    try { $all = $AE::RootElement.FindAll([Windows.Automation.TreeScope]::Descendants, $pidCond) } catch { continue }
    foreach ($e in $all) {
        $n = $e.Current.Name
        if (-not $pressed -and $e.Current.ControlType -eq [Windows.Automation.ControlType]::Button -and $n -eq 'Recover') {
            $e.GetCurrentPattern([Windows.Automation.InvokePattern]::Pattern).Invoke()
            $pressed = $true
        }
        if ($pressed -and $n -and $n.Length -gt 3 -and -not $lines.Contains($n)) { $lines[$n] = $true }
    }
    if ($pressed -and ((Get-Date) - $start).TotalSeconds -gt 60) { break }
}
$lines.Keys | Where-Object { $_ -match 'error|invalid|fix|recover|audit|found|Pass|bad|AcDb' }
Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
