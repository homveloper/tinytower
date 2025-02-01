@echo off
setlocal

echo Building TinyTower for Windows, macOS, and Linux...

REM Create the build directory if it doesn't exist
if not exist build (
    mkdir build
)

REM Build for Windows
echo Building for Windows...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -o build\tinytower-windows.exe main.go
if errorlevel 1 goto error

REM Build for Linux
echo Building for Linux...
set GOOS=linux
set GOARCH=amd64
go build -o build\tinytower-linux main.go
if errorlevel 1 goto error

echo All builds complete! Check the "build" folder for your executables.
goto end

:error
echo Build failed!
:end
pause
endlocal
