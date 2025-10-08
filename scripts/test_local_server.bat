@echo off
setlocal enabledelayedexpansion

REM Default options
set "HOST_URL=http://127.0.0.1:8080"
set "TOKEN=dev-token"

if /I "%~1"=="-h" goto :usage
if /I "%~1"=="--help" goto :usage

:parse
if "%~1"=="" goto :after_parse
if /I "%~1"=="--host" (
  if "%~2"=="" goto :usage
  set "HOST_URL=%~2"
  shift
  shift
  goto :parse
)
if /I "%~1"=="--token" (
  if "%~2"=="" goto :usage
  set "TOKEN=%~2"
  shift
  shift
  goto :parse
)
echo Unknown arg: %~1
goto :usage

:usage
echo Usage: scripts\test_local_server.bat [--host URL] [--token TOKEN]
echo.
echo Runs a local test against a running CoderForge API server:
echo   - Builds the dev provider
echo   - Uses .terraformrc dev override
echo   - For each example: init, validate, plan, apply, output, destroy
echo.
echo Options:
echo   --host   Local API base URL ^(default: http://127.0.0.1:8080^)
echo   --token  API token to use   ^(default: dev-token^)
exit /b 1

:after_parse

REM Resolve repo root: prefer git, else script dir parent, else CWD
set "ROOT_DIR="
where git >nul 2>nul
if not errorlevel 1 (
  for /f "usebackq delims=" %%G in (`git rev-parse --show-toplevel 2^>nul`) do set "ROOT_DIR=%%G"
)
if "%ROOT_DIR%"=="" (
  set "SCRIPT_DIR=%~dp0"
  if "%SCRIPT_DIR:~-1%"=="\" set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"
  for %%I in ("%SCRIPT_DIR%\..") do set "ROOT_DIR=%%~fI"
)
if not exist "%ROOT_DIR%\go.mod" (
  set "ROOT_DIR=%CD%"
)
if not exist "%ROOT_DIR%\go.mod" (
  echo ERROR: could not locate go.mod. Run this script from within the repository.
  exit /b 3
)

REM Check dependencies
where go >nul 2>nul
if errorlevel 1 (
  echo ERROR: go is required on PATH
  exit /b 2
)
where terraform >nul 2>nul
if errorlevel 1 (
  echo ERROR: terraform is required on PATH
  exit /b 2
)

REM Environment for Terraform CLI
set "TF_CLI_CONFIG_FILE=%ROOT_DIR%\.terraformrc"
set "CODERFORGE_API_URL=%HOST_URL%"
set "CODERFORGE_CLOUD_TOKEN=%TOKEN%"

echo Building dev provider ...
pushd "%ROOT_DIR%" >nul
if not exist registry.terraform.io mkdir registry.terraform.io >nul 2>nul
if not exist registry.terraform.io\coderforge mkdir registry.terraform.io\coderforge >nul 2>nul
go build -o registry.terraform.io\coderforge\coderforge
if errorlevel 1 (
  popd >nul
  exit /b 1
)
popd >nul

call :run_example examples\resources\function
if errorlevel 1 exit /b 1
call :run_example examples\resources\container_registry
if errorlevel 1 exit /b 1

echo.
echo All local tests completed successfully.
exit /b 0

:run_example
set "EX_DIR=%~1"
echo.
echo === Testing example: %EX_DIR% ===
pushd "%ROOT_DIR%\%EX_DIR%" >nul
terraform init -input=false -upgrade
if errorlevel 1 (
  popd >nul
  exit /b 1
)
terraform validate
if errorlevel 1 (
  popd >nul
  exit /b 1
)
terraform plan -out=tfplan
if errorlevel 1 (
  popd >nul
  exit /b 1
)
terraform apply -auto-approve tfplan
if errorlevel 1 (
  popd >nul
  exit /b 1
)
terraform output -json >nul 2>nul
terraform destroy -auto-approve
if errorlevel 1 (
  echo WARN: destroy failed; attempting re-run once
  terraform destroy -auto-approve >nul 2>nul
)
popd >nul
exit /b 0

