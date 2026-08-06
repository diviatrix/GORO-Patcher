@echo off
setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
set "SRC_DIR=%PROJECT_DIR%\src"
set "OUT_DIR=%PROJECT_DIR%\build"

echo [build] GORO-Patcher build script (Windows)

if "%1"=="-c" (
    echo [build] Cleaning...
    if exist "%OUT_DIR%\GORO-Patcher.exe" del "%OUT_DIR%\GORO-Patcher.exe"
    if exist "%OUT_DIR%\hashfile.exe" del "%OUT_DIR%\hashfile.exe"
    if exist "%SRC_DIR%\build" rmdir /s /q "%SRC_DIR%\build"
    shift
)

echo [build] Copying frontend...
copy /y "%SRC_DIR%\frontend\src\main.js" "%SRC_DIR%\frontend\dist\" >nul
xcopy /s /e /y "%SRC_DIR%\frontend\bindings" "%SRC_DIR%\frontend\dist\bindings\" >nul

if "%1"=="-f" (
    echo [build] Generating bindings...
    cd /d "%SRC_DIR%"
    call wails3 generate bindings -b
    xcopy /s /e /y "%SRC_DIR%\frontend\bindings" "%SRC_DIR%\frontend\dist\bindings\" >nul
    shift
)

echo [build] Building hashfile...
cd /d "%SRC_DIR%"
set CGO_ENABLED=0
go build -o "%OUT_DIR%\hashfile.exe" .\cmd\hashfile
if errorlevel 1 (
    echo [build] ERROR: hashfile build failed
    exit /b 1
)

echo [build] Building GORO-Patcher...
set CGO_ENABLED=1
call wails3 build
if errorlevel 1 (
    echo [build] ERROR: patcher build failed
    exit /b 1
)

copy /y "%SRC_DIR%\build\GORO-Patcher.exe" "%OUT_DIR%\GORO-Patcher.exe" >nul
rmdir /s /q "%SRC_DIR%\build"

echo [build] Copying example files...
if not exist "%OUT_DIR%\public\data" mkdir "%OUT_DIR%\public\data"
copy /y "%PROJECT_DIR%\example\plist.json.example" "%OUT_DIR%\public\" >nul
if exist "%OUT_DIR%\goro-config.json.example" copy /y "%OUT_DIR%\goro-config.json.example" "%OUT_DIR%\" >nul

echo [build] Done!
dir /b "%OUT_DIR%\*.exe" "%OUT_DIR%\*.example"
