@echo off
cd /d "%~dp0"
if exist "%~dp0wingman.exe" (
  "%~dp0wingman.exe" %*
) else (
  go run ./cmd/wingman %*
)
