@echo off
setlocal EnableExtensions EnableDelayedExpansion
rem DeepTalk Windows 构建脚本，与 Makefile 行为保持一致。

set "APP=deeptalk"
set "BIN_DIR=bin"
set "CGO=0"
set "GOPROXY=https://goproxy.cn,direct"
set "GOSUMDB=sum.golang.google.cn"
set "GOCACHE=%~dp0..\.gocache"
set "GOMODCACHE=%~dp0..\.gomodcache"
set "BUILD_TIME=unknown"

set "VERSION=v1.0.0"
for /f "delims=" %%v in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%v"

set "COMMIT=unknown"
for /f "delims=" %%v in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%v"

set "LDFLAGS=-s -w -X main.version=%VERSION% -X main.commit=%COMMIT% -X main.buildTime=%BUILD_TIME%"

if "%~1"=="" goto build
if /i "%~1"=="build" goto build
if /i "%~1"=="build-linux" goto build_linux
if /i "%~1"=="build-linux-arm64" goto build_linux_arm64
if /i "%~1"=="build-windows" goto build_windows
if /i "%~1"=="build-darwin" goto build_darwin
if /i "%~1"=="build-darwin-amd64" goto build_darwin_amd64
if /i "%~1"=="build-all" goto build_all
if /i "%~1"=="run" goto run
if /i "%~1"=="test" goto test
if /i "%~1"=="fmt" goto fmt
if /i "%~1"=="vet" goto vet
if /i "%~1"=="tidy" goto tidy
if /i "%~1"=="ui-build" goto ui_build
if /i "%~1"=="embed" goto embed
if /i "%~1"=="dist" goto dist
if /i "%~1"=="clean" goto clean

echo Unknown target: %~1
exit /b 1

:build
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
set "CGO_ENABLED=%CGO%"
set "GOOS="
set "GOARCH="
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN_DIR%\%APP%.exe" .
if errorlevel 1 exit /b 1
exit /b 0

:build_linux
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
set "CGO_ENABLED=%CGO%"
set "GOOS=linux"
set "GOARCH=amd64"
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN_DIR%\%APP%-linux-amd64" .
exit /b %errorlevel%

:build_linux_arm64
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
set "CGO_ENABLED=%CGO%"
set "GOOS=linux"
set "GOARCH=arm64"
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN_DIR%\%APP%-linux-arm64" .
exit /b %errorlevel%

:build_windows
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
set "CGO_ENABLED=%CGO%"
set "GOOS=windows"
set "GOARCH=amd64"
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN_DIR%\%APP%-windows-amd64.exe" .
exit /b %errorlevel%

:build_darwin
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
set "CGO_ENABLED=%CGO%"
set "GOOS=darwin"
set "GOARCH=arm64"
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN_DIR%\%APP%-darwin-arm64" .
exit /b %errorlevel%

:build_darwin_amd64
if not exist "%BIN_DIR%" mkdir "%BIN_DIR%"
set "CGO_ENABLED=%CGO%"
set "GOOS=darwin"
set "GOARCH=amd64"
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN_DIR%\%APP%-darwin-amd64" .
exit /b %errorlevel%

:build_all
call "%~f0" build-linux
call "%~f0" build-linux-arm64
call "%~f0" build-windows
call "%~f0" build-darwin
call "%~f0" build-darwin-amd64
exit /b 0

:run
go run .
exit /b %errorlevel%

:test
go test ./...
exit /b %errorlevel%

:fmt
gofmt -w .
exit /b 0

:vet
go vet ./...
exit /b %errorlevel%

:tidy
go mod tidy
exit /b %errorlevel%

:ui_build
pushd "%~dp0..\ui"
call pnpm install
if errorlevel 1 (popd & exit /b 1)
call pnpm build
if errorlevel 1 (popd & exit /b 1)
popd
exit /b 0

:embed
call "%~f0" ui-build
if errorlevel 1 exit /b 1
if exist "web\dist\c" rmdir /s /q "web\dist\c"
if exist "web\dist\b" rmdir /s /q "web\dist\b"
mkdir "web\dist\c" 2>nul
mkdir "web\dist\b" 2>nul
xcopy /e /i /y "..\ui\c\dist\*" "web\dist\c\" >nul
xcopy /e /i /y "..\ui\b\dist\*" "web\dist\b\" >nul
exit /b 0

:dist
call "%~f0" embed
if errorlevel 1 exit /b 1
call "%~f0" build
exit /b %errorlevel%

:clean
if exist "%BIN_DIR%" rmdir /s /q "%BIN_DIR%"
exit /b 0
