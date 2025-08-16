$exeUrl = "https://github.com/nguyendangkin/chin-pak/releases/download/v1.0.0/chin.exe"
$installDir = "$env:ProgramFiles\Chin"
$exePath = Join-Path $installDir "chin.exe"

if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Write-Host "Dang tai chin.exe..."
Invoke-WebRequest -Uri $exeUrl -OutFile $exePath

$target = "Machine"
$oldPath = [Environment]::GetEnvironmentVariable("Path", $target)
if ($oldPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$oldPath;$installDir", $target)
    Write-Host "Da them vao PATH."
}

Write-Host "Cai dat thanh cong! Hay mo lai terminal va go: chin"
