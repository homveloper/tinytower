@echo off
setlocal

echo ====================================================
echo Building TinyTower for various Windows architectures...
echo ====================================================

REM Create the build directory if it doesn't exist
if not exist build (
    mkdir build
)

echo.
echo Building for Windows x86 (386)...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=386
go build -o build\tinytower-windows-386.exe main.go
if errorlevel 1 goto error

echo.
echo Building for Windows x64 (amd64)...
set GOARCH=amd64
go build -o build\tinytower-windows-amd64.exe main.go
if errorlevel 1 goto error

echo.
echo Building for Windows ARM (arm)...
set GOARCH=arm
go build -o build\tinytower-windows-arm.exe main.go
if errorlevel 1 goto error

echo.
echo Building for Windows ARM64 (arm64)...
set GOARCH=arm64
go build -o build\tinytower-windows-arm64.exe main.go
if errorlevel 1 goto error

echo.
echo ====================================================
echo All Windows builds complete! Check the "build" folder.
echo ====================================================
goto end

:error
echo.
echo Build failed!
:end
pause
endlocal
