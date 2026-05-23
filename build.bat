@echo off
cd /d "%~dp0"
go build -ldflags="-s -w" -o wingman.exe ./cmd/wingman
if %ERRORLEVEL% neq 0 exit /b %ERRORLEVEL%
echo Built wingman.exe
