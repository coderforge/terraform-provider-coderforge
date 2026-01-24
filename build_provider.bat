@echo off
setlocal

:: --- Configuration ---
set "VERSION=1.1.0"
set "PROVIDER_NAME=coderforge"
set "NAMESPACE=coderforge"
set "HOSTNAME=registry.terraform.io"
set "OS_ARCH=windows_amd64"

:: --- Directories ---
set "TARGET_DIR=%APPDATA%\terraform.d\plugins\%HOSTNAME%\%NAMESPACE%\%PROVIDER_NAME%\%VERSION%\%OS_ARCH%"
set "TF_RC_FILE=%APPDATA%\terraform.rc"

echo [1/3] Building Terraform Provider...
go build -o terraform-provider-%PROVIDER_NAME%_v%VERSION%.exe
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Build failed.
    exit /b %ERRORLEVEL%
)

echo [2/3] Installing provider to local mirror...
if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%"
move /Y "terraform-provider-%PROVIDER_NAME%_v%VERSION%.exe" "%TARGET_DIR%\" >nul

echo [3/3] Configuring terraform.rc (Generating safe paths)...

:: Use PowerShell to calculate the path with forward slashes to avoid Escape Sequence errors
for /f "delims=" %%I in ('powershell -Command "'%APPDATA%\terraform.d\plugins'.Replace('\', '/') "') do set "SAFE_PATH=%%I"

(
echo provider_installation {
echo   filesystem_mirror {
echo     path    = "%SAFE_PATH%"
echo     include = ["registry.terraform.io/coderforge/coderforge"]
echo   }
echo   direct {
echo     exclude = ["registry.terraform.io/coderforge/coderforge"]
echo   }
echo }
) > "%TF_RC_FILE%"

echo.
echo ==========================================
echo  Build Success!
echo  Provider Path: %SAFE_PATH%
echo ==========================================
pause