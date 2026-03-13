@echo off
setlocal

set ROOT_DIR=%~dp0..
for %%I in ("%ROOT_DIR%") do set ROOT_DIR=%%~fI
set DIST_DIR=%ROOT_DIR%\dist

set PACKAGE_NAME=%1
if "%PACKAGE_NAME%"=="" set PACKAGE_NAME=clipsync-windows-amd64

set STAGE_DIR=%DIST_DIR%\%PACKAGE_NAME%
set ZIP_PATH=%DIST_DIR%\%PACKAGE_NAME%.zip

if not exist "%DIST_DIR%" mkdir "%DIST_DIR%"
if exist "%STAGE_DIR%" rmdir /s /q "%STAGE_DIR%"
if exist "%ZIP_PATH%" del /q "%ZIP_PATH%"

set GOOS=windows
set GOARCH=amd64

go build -trimpath -ldflags "-s -w" -o "%DIST_DIR%\clipsync-server.exe" "%ROOT_DIR%\cmd\server"
if errorlevel 1 exit /b 1
go build -trimpath -ldflags "-s -w" -o "%DIST_DIR%\clipsync-client.exe" "%ROOT_DIR%\cmd\client"
if errorlevel 1 exit /b 1

mkdir "%STAGE_DIR%"
mkdir "%STAGE_DIR%\configs"
mkdir "%STAGE_DIR%\scripts"

copy /y "%DIST_DIR%\clipsync-server.exe" "%STAGE_DIR%\" >nul
copy /y "%DIST_DIR%\clipsync-client.exe" "%STAGE_DIR%\" >nul
copy /y "%ROOT_DIR%\configs\server.json" "%STAGE_DIR%\configs\" >nul
copy /y "%ROOT_DIR%\configs\client.json" "%STAGE_DIR%\configs\" >nul
copy /y "%ROOT_DIR%\README.md" "%STAGE_DIR%\" >nul
if exist "%ROOT_DIR%\LICENSE" copy /y "%ROOT_DIR%\LICENSE" "%STAGE_DIR%\" >nul
copy /y "%ROOT_DIR%\scripts\install-autostart-client.ps1" "%STAGE_DIR%\scripts\" >nul
copy /y "%ROOT_DIR%\scripts\install-autostart-server.ps1" "%STAGE_DIR%\scripts\" >nul
copy /y "%ROOT_DIR%\scripts\uninstall-autostart.ps1" "%STAGE_DIR%\scripts\" >nul

powershell -NoProfile -ExecutionPolicy Bypass -Command "Compress-Archive -Path '%STAGE_DIR%' -DestinationPath '%ZIP_PATH%' -Force"
if errorlevel 1 exit /b 1

echo Package ready: %ZIP_PATH%
echo Unpacked root: %STAGE_DIR%