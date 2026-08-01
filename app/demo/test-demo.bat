@echo off
setlocal enabledelayedexpansion

rem Parallel smoke test for the media API (snail + squirrel).
rem Usage: test-demo.bat [operation]
rem Defaults to grayscale. Operations: convert, resize, compress, grayscale
rem Polls both sessions every 10s.

set "BASE=http://media-dev-alb-1228361591.eu-central-1.elb.amazonaws.com"
set "V1=%~dp0snail.mp4"
set "V2=%~dp0squirrel.mp4"
set "OP=%~1"
if "%OP%"=="" set "OP=grayscale"

rem expected processing time (seconds) for the larger squirrel.mp4 (1.2MB)
set "EXPECTED=30"
if /i "%OP%"=="convert"   set "EXPECTED=30"
if /i "%OP%"=="resize"    set "EXPECTED=30"
if /i "%OP%"=="compress"  set "EXPECTED=20"
if /i "%OP%"=="grayscale" set "EXPECTED=30"

set "POLL_INTERVAL=10"
set "MAX_ITER=30"
set "OUT=%~dp0output"
if not exist "%OUT%" mkdir "%OUT%"

echo == media smoke test (parallel) ==
echo operation:    %OP%
echo video 1:      snail.mp4 (0.7MB)
echo video 2:      squirrel.mp4 (1.2MB)
echo expected ~%EXPECTED%s to complete, polling every %POLL_INTERVAL%s
echo.

echo == upload snail ==
curl -s -F "file=@%V1%" "%BASE%/upload" > "%~dp0.tmp_upload1.json"
type "%~dp0.tmp_upload1.json"
echo.

echo == upload squirrel ==
curl -s -F "file=@%V2%" "%BASE%/upload" > "%~dp0.tmp_upload2.json"
type "%~dp0.tmp_upload2.json"
echo.

for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "(Get-Content '%~dp0.tmp_upload1.json' -Raw | ConvertFrom-Json).session_id"`) do set "SID1=%%i"
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "(Get-Content '%~dp0.tmp_upload2.json' -Raw | ConvertFrom-Json).session_id"`) do set "SID2=%%i"
echo snail:     %SID1%
echo squirrel:  %SID2%

if "%SID1%"=="" ( echo ERROR: failed to parse snail session_id & goto CLEANFAIL )
if "%SID2%"=="" ( echo ERROR: failed to parse squirrel session_id & goto CLEANFAIL )

echo == apply %OP% ==
curl -s -X POST "%BASE%/sessions/%SID1%/apply" -H "Content-Type: application/json" -d "{\"operation\":\"%OP%\"}"
echo.
curl -s -X POST "%BASE%/sessions/%SID2%/apply" -H "Content-Type: application/json" -d "{\"operation\":\"%OP%\"}"
echo.

echo == polling both every %POLL_INTERVAL%s (expected ~%EXPECTED%s) ==
set /a SECONDS=0
set /a ITER=0
:POLL
set /a SECONDS=%SECONDS%+%POLL_INTERVAL%
set /a ITER+=1
if !ITER! gtr %MAX_ITER% (
  echo ERROR: timeout after %MAX_ITER%*%POLL_INTERVAL%s
  goto CLEANFAIL
)
set "S1="
set "S2="
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "$r=(Invoke-RestMethod -Uri '%BASE%/sessions/%SID1%/status'); $r.job.status + ' v' + $r.current_version"`) do set "S1=%%i"
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "$r=(Invoke-RestMethod -Uri '%BASE%/sessions/%SID2%/status'); $r.job.status + ' v' + $r.current_version"`) do set "S2=%%i"
echo [%SECONDS%s] snail: !S1!   squirrel: !S2!
set "DONE1="
set "DONE2="
echo !S1! | findstr /c:"done" >nul && set "DONE1=1"
echo !S1! | findstr /c:"failed" >nul && set "DONE1=1"
echo !S2! | findstr /c:"done" >nul && set "DONE2=1"
echo !S2! | findstr /c:"failed" >nul && set "DONE2=1"
if "!DONE1!!DONE2!"=="11" goto DONE
ping -n 11 127.0.0.1 >nul
goto POLL

:DONE
echo.
echo == snail download ==
curl -s "%BASE%/sessions/%SID1%/download" > "%~dp0.tmp_url1.json"
type "%~dp0.tmp_url1.json"
echo.
echo == squirrel download ==
curl -s "%BASE%/sessions/%SID2%/download" > "%~dp0.tmp_url2.json"
type "%~dp0.tmp_url2.json"
echo.
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "(Get-Content '%~dp0.tmp_url1.json' -Raw | ConvertFrom-Json).download_url"`) do set "URL1=%%i"
for /f "usebackq delims=" %%i in (`powershell -NoProfile -Command "(Get-Content '%~dp0.tmp_url2.json' -Raw | ConvertFrom-Json).download_url"`) do set "URL2=%%i"
echo.
echo == saving to %OUT% ==
curl -s -o "%OUT%\snail-%OP%.mp4" "%URL1%"
curl -s -o "%OUT%\squirrel-%OP%.mp4" "%URL2%"
del "%~dp0.tmp_upload1.json" "%~dp0.tmp_upload2.json" "%~dp0.tmp_url1.json" "%~dp0.tmp_url2.json" 2>nul
for %%f in ("%OUT%\snail-%OP%.mp4" "%OUT%\squirrel-%OP%.mp4") do echo saved: %%~f (%%~zf bytes)
echo == done. sessions %SID1% / %SID2% ==
endlocal
exit /b 0

:CLEANFAIL
del "%~dp0.tmp_upload1.json" "%~dp0.tmp_upload2.json" "%~dp0.tmp_url1.json" "%~dp0.tmp_url2.json" 2>nul
exit /b 1
