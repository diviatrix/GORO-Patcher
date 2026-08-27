@echo off
setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
set "SRC_DIR=%PROJECT_DIR%\src"
set "OUT_DIR=%PROJECT_DIR%\build"

set "RELEASE="
if "%1"=="-r" set "RELEASE=release"
if "%1"=="-c" (
    echo [build] Cleaning...
    if exist "%OUT_DIR%\GORO-Patcher.exe" del "%OUT_DIR%\GORO-Patcher.exe"
    if exist "%OUT_DIR%\hashfile.exe" del "%OUT_DIR%\hashfile.exe"
    if exist "%SRC_DIR%\wails_windows_amd64.syso" del "%SRC_DIR%\wails_windows_amd64.syso"
)

echo [build] GORO-Patcher build script (Windows) %RELEASE%

echo [build] Generating TypeScript bindings...
cd /d "%SRC_DIR%"
call wails3 generate bindings -b -d frontend/dist/bindings
if errorlevel 1 (
    echo [build] ERROR: bindings generation failed
    exit /b 1
)

echo [build] Generating Windows .syso (icon/manifest)...
cd /d "%SRC_DIR%\windows"
call wails3 generate syso -arch amd64 -icon icon.ico -manifest wails.exe.manifest -info info.json -out ..\wails_windows_amd64.syso
if errorlevel 1 (
    echo [build] ERROR: syso generation failed
    exit /b 1
)
cd /d "%SRC_DIR%"

set "TAGS=production"
if "%RELEASE%"=="release" set "TAGS=production,release"

echo [build] Building hashfile...
set CGO_ENABLED=0
go build -trimpath -buildvcs=false -ldflags="-w -s" -o "%OUT_DIR%\hashfile.exe" .\cmd\hashfile
if errorlevel 1 (
    echo [build] ERROR: hashfile build failed
    exit /b 1
)

echo [build] Building GORO-Patcher (%TAGS%)...
set CGO_ENABLED=0
go build -tags "%TAGS%" -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui" -o "%OUT_DIR%\GORO-Patcher.exe" .
if errorlevel 1 (
    echo [build] ERROR: patcher build failed
    exit /b 1
)

echo [build] Copying example files...
if not exist "%OUT_DIR%\public\data" mkdir "%OUT_DIR%\public\data"
copy /y "%PROJECT_DIR%\example\plist.json.example" "%OUT_DIR%\public\" >nul
if exist "%PROJECT_DIR%\example\goro-config.json.example" copy /y "%PROJECT_DIR%\example\goro-config.json.example" "%OUT_DIR%\" >nul

echo [build] Done!
dir /b "%OUT_DIR%\GORO-Patcher.exe" "%OUT_DIR%\hashfile.exe"