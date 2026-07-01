param(
    [Parameter(Mandatory=$true)]
    [string]$InputDir,
    [Parameter(Mandatory=$true)]
    [string]$OutputFile,
    [int]$MinZoom = 5,
    [int]$MaxZoom = 17
)

$ErrorActionPreference = "Stop"

# Check dependencies
function Check-Cmd {
    param($Name, $InstallHint)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Error "需要 $Name — $InstallHint"
        exit 1
    }
}

Check-Cmd "gdalbuildvrt" "pip install gdal"
Check-Cmd "rio" "pip install rio-pmtiles"

$ecwFiles = Get-ChildItem -Path $InputDir -Recurse -Filter "*.ecw" | Select-Object -ExpandProperty FullName
if ($ecwFiles.Count -eq 0) {
    Write-Error "目錄 $InputDir 中找不到 *.ecw 檔案"
    exit 1
}

Write-Host "找到 $($ecwFiles.Count) 個 ECW 檔案" -ForegroundColor Green

# Step 1: VRT
$vrtPath = [System.IO.Path]::GetTempFileName() + ".vrt"
Write-Host "步驟 1/3: 建立 VRT → $vrtPath" -ForegroundColor Cyan
$vrtArgs = @($ecwFiles -join " ", "-o", $vrtPath)
Start-Process -Wait -NoNewWindow -FilePath "gdalbuildvrt" -ArgumentList $vrtArgs
if (-not (Test-Path $vrtPath)) {
    Write-Error "VRT 建立失敗"
    exit 1
}

# Step 2: Build overviews (for faster zoom-out)
Write-Host "步驟 2/3: 建立概觀圖層" -ForegroundColor Cyan
Start-Process -Wait -NoNewWindow -FilePath "gdaladdo" -ArgumentList @($vrtPath, "2", "4", "8", "16")

# Step 3: Convert to PMTiles
Write-Host "步驟 3/3: 轉換為 PMTiles → $OutputFile" -ForegroundColor Cyan
$rioArgs = @("pmtiles", $vrtPath, $OutputFile, "--zoom-levels", "$MinZoom..$MaxZoom")
Start-Process -Wait -NoNewWindow -FilePath "rio" -ArgumentList $rioArgs

# Cleanup
Remove-Item $vrtPath -Force -ErrorAction SilentlyContinue

if (Test-Path $OutputFile) {
    $size = (Get-Item $OutputFile).Length / 1GB
    Write-Host "✓ 完成！$OutputFile ($([math]::Round($size, 2)) GB)" -ForegroundColor Green
} else {
    Write-Error "轉換失敗"
    exit 1
}
