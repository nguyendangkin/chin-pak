$exeUrl = "https://github.com/nguyendangkin/chin-pak/releases/download/v1.0.1/chin.exe"
$installDir = "$env:ProgramFiles\Chin"
$exePath = Join-Path $installDir "chin.exe"

if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Write-Host "Downloading chin.exe..."
Invoke-WebRequest -Uri $exeUrl -OutFile $exePath

$target = "Machine"
$oldPath = [Environment]::GetEnvironmentVariable("Path", $target)
if ($oldPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$oldPath;$installDir", $target)
    Write-Host "Added to the PATH."
}

Write-Host "Successfully installed, please reopen Terminal and type: chin"
