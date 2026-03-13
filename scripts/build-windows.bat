@echo off
setlocal

if "%GOOS%"=="" set GOOS=windows
if "%GOARCH%"=="" set GOARCH=amd64

if not exist dist mkdir dist

echo Building ClipSync server...
go build -trimpath -ldflags "-s -w" -o dist\clipsync-server.exe .\cmd\server
if errorlevel 1 exit /b 1

echo Building ClipSync client...
go build -trimpath -ldflags "-s -w" -o dist\clipsync-client.exe .\cmd\client
if errorlevel 1 exit /b 1

echo Build complete. Files are in dist\
