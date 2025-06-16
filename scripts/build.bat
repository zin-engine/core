@echo off
setlocal EnableDelayedExpansion

echo [ZIN] Building Zin...

REM Function to convert time to hundredths of a second
call :getTimeInHundredths startTime

REM Set target OS and architecture
set GOOS=windows
set GOARCH=amd64

REM Create .build directory if not exists
if not exist ..\.build (
    mkdir ..\.build
)

REM Clean previous binary
if exist ..\.build\zin.exe (
    del ..\.build\zin.exe
)

REM Build the binary
go build -o ./../.build/zin.exe ./../cmd/zin

REM Get end time
call :getTimeInHundredths endTime

REM Calculate duration
set /A durationMs = !endTime! - !startTime!

REM Handle midnight wrap
if !durationMs! lss 0 (
    set /A durationMs += 8640000
)

REM Convert to seconds
set /A seconds = durationMs / 100
set /A hundredths = durationMs %% 100

REM Show result
if %errorlevel%==0 (
    echo [ZIN] Build successful! zin.exe is in the /.build folder.
) else (
    echo [ZIN] Build failed.
)

echo [ZIN] Build time: !seconds!.!hundredths! sec

endlocal
exit /b

:getTimeInHundredths <returnVar>
    for /f "tokens=1-4 delims=:.," %%a in ("%time: =0%") do (
        set /A "h=1%%a-100, m=1%%b-100, s=1%%c-100, hs=1%%d-100"
        set /A "%~1=((h*60 + m)*60 + s)*100 + hs"
    )
    exit /b
