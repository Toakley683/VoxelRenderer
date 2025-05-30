@echo off

set GOTMPDIR=C:\GoTemp
if not exist "%GOTMPDIR%" mkdir "%GOTMPDIR%"

go build .
