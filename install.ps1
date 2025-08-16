$exeUrl = "https://github.com/nguyendangkin/chin-pak/releases/download/v1.0.0/chin.exe"
$installDir = "$env:ProgramFiles\Chin"
$exePath = Join-Path $installDir "chin.exe"

# Tạo folder cài đặt
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

# Tải exe từ GitHub release
Write-Host "Dang tai chin.exe..."
Invoke-WebRequest -Uri $exeUrl -OutFile $exePath

# Thêm vào PATH nếu chưa có
$target = "Machine"  # hoặc "User" nếu chỉ muốn cho user hiện tại
$oldPath = [Environment]::GetEnvironmentVariable("Path", $target)
if ($oldPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$oldPath;$installDir", $target)
    Write-Host "Da them vao PATH."
}

Write-Host "Cai dat thanh cong! Hay mo lai terminal va go: chin"
