# build.ps1 — TCP Tester 交叉编译脚本
# 用法: .\build.ps1 [目标]
# 目标: all(默认), gui, cli-win, cli-linux

param(
    [string]$Target = "all"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = $PSScriptRoot
$OutputDir = Join-Path $ProjectRoot "build\bin"

# 确保输出目录存在
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

function Write-Header($text) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host " $text" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
}

# 检测 go-winres 是否已安装，未安装则自动安装
function Ensure-GoWinres {
    if (!(Get-Command "go-winres" -ErrorAction SilentlyContinue)) {
        Write-Host "正在安装 go-winres..." -ForegroundColor Yellow
        go install github.com/tc-hib/go-winres@latest
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ go-winres 安装失败，请手动运行: go install github.com/tc-hib/go-winres@latest" -ForegroundColor Red
            exit 1
        }
        Write-Host "✅ go-winres 安装成功" -ForegroundColor Green
    }
}

# 清理 go-winres 生成的临时文件
function Cleanup-Winres {
    Set-Location $ProjectRoot
    Remove-Item "rsrc_windows_*.syso" -ErrorAction SilentlyContinue
}

function Build-GUI {
    Write-Header "构建 Windows GUI (Wails)"
    Set-Location $ProjectRoot
    wails build
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Windows GUI 构建成功: build\bin\tcp-tester.exe" -ForegroundColor Green
    } else {
        Write-Host "❌ Windows GUI 构建失败" -ForegroundColor Red
        exit 1
    }
}

function Build-CLI-Windows {
    Write-Header "构建 Windows CLI"
    Ensure-GoWinres
    Set-Location $ProjectRoot

    # 使用 go-winres 嵌入 manifest（requireAdministrator）
    go-winres make --product-version "1.0.0" --file-version "1.0.0"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ go-winres 嵌入 manifest 失败" -ForegroundColor Red
        Cleanup-Winres
        exit 1
    }

    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    $output = Join-Path $OutputDir "tcp-tester-cli.exe"
    go build -tags cli -ldflags="-s -w" -o $output
    if ($LASTEXITCODE -eq 0) {
        $size = [math]::Round((Get-Item $output).Length / 1MB, 2)
        Write-Host "✅ Windows CLI 构建成功: $output ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Windows CLI 构建失败" -ForegroundColor Red
        Cleanup-Winres
        exit 1
    }

    Cleanup-Winres
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}

function Build-CLI-Linux {
    Write-Header "构建 Linux CLI"
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    $output = Join-Path $OutputDir "tcp-tester-linux-amd64"
    Set-Location $ProjectRoot
    go build -tags cli -ldflags="-s -w" -o $output
    if ($LASTEXITCODE -eq 0) {
        $size = [math]::Round((Get-Item $output).Length / 1MB, 2)
        Write-Host "✅ Linux CLI 构建成功: $output ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Linux CLI 构建失败" -ForegroundColor Red
        exit 1
    }
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}

function Build-CLI-LinuxArm64 {
    Write-Header "构建 Linux CLI (ARM64)"
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    $output = Join-Path $OutputDir "tcp-tester-linux-arm64"
    Set-Location $ProjectRoot
    go build -tags cli -ldflags="-s -w" -o $output
    if ($LASTEXITCODE -eq 0) {
        $size = [math]::Round((Get-Item $output).Length / 1MB, 2)
        Write-Host "✅ Linux ARM64 CLI 构建成功: $output ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Linux ARM64 CLI 构建失败" -ForegroundColor Red
        exit 1
    }
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}

# 主逻辑
Write-Host "TCP Tester 交叉编译脚本" -ForegroundColor Yellow
Write-Host "目标: $Target" -ForegroundColor Yellow

switch ($Target) {
    "all" {
        Build-GUI
        Build-CLI-Windows
        Build-CLI-Linux
        Build-CLI-LinuxArm64
    }
    "gui" {
        Build-GUI
    }
    "cli-win" {
        Build-CLI-Windows
    }
    "cli-linux" {
        Build-CLI-Linux
    }
    "cli-linux-arm64" {
        Build-CLI-LinuxArm64
    }
    "cli" {
        Build-CLI-Windows
        Build-CLI-Linux
    }
    default {
        Write-Host "未知目标: $Target" -ForegroundColor Red
        Write-Host ""
        Write-Host "用法: .\build.ps1 [目标]"
        Write-Host ""
        Write-Host "可用目标:"
        Write-Host "  all              构建所有版本 (默认)"
        Write-Host "  gui              仅构建 Windows GUI"
        Write-Host "  cli-win          仅构建 Windows CLI"
        Write-Host "  cli-linux        仅构建 Linux CLI (amd64)"
        Write-Host "  cli-linux-arm64  仅构建 Linux CLI (arm64)"
        Write-Host "  cli              构建所有 CLI 版本"
        exit 1
    }
}

Write-Host ""
Write-Header "构建完成"
Write-Host "输出目录: $OutputDir" -ForegroundColor Yellow
Get-ChildItem $OutputDir -File | ForEach-Object {
    $size = [math]::Round($_.Length / 1MB, 2)
    Write-Host "  $($_.Name) ($size MB)" -ForegroundColor White
}
