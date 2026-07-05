# Generate barcode PNGs with the barcode-rest CLI (no server needed).
# Usage: .\cli_example.ps1 [-Exe path\to\barcode-rest.exe]

param(
    [string]$Exe = (Join-Path $PSScriptRoot "..\barcode-rest.exe")
)

$outDir = Join-Path $PSScriptRoot "cli_output"
New-Item -ItemType Directory -Force $outDir | Out-Null

$jobs = @(
    # 2D: exact 256x256 output, module size chosen automatically
    @("datamatrix", "--text", "ABC123", "--size", "256", "--output", "$outDir\datamatrix.png"),
    # QR with error correction level Q
    @("qr", "--text", "https://example.com/lot/ABC123", "--level", "Q", "--size", "512", "--output", "$outDir\qr.png"),
    # 1D with a human-readable label below the bars
    @("code128", "--text", "ABC-123456", "--label", "--output", "$outDir\code128.png"),
    # EAN-13 (JAN): 12 digits in, check digit computed automatically
    @("ean13", "--text", "490123456789", "--label", "--output", "$outDir\ean13.png")
)

foreach ($job in $jobs) {
    & $Exe generate @job
    if ($LASTEXITCODE -ne 0) {
        Write-Error "generate $($job[0]) failed (exit $LASTEXITCODE)"
        exit 1
    }
    Write-Host "OK  $($job[0])"
}

Write-Host "Saved PNGs to $outDir"
