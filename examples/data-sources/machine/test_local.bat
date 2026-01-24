@echo off
setlocal

:: PULIZIA FORZATA DELLA CACHE
echo [INFO] Cleaning Terraform cache...
if exist ".terraform" rd /s /q ".terraform"
if exist ".terraform.lock.hcl" del ".terraform.lock.hcl"

echo.
echo [INFO] Initializing Terraform...
:: Rimuovi -upgrade perché stiamo partendo da zero
terraform init

echo.
echo [INFO] Planning infrastructure...
terraform plan

echo.
echo [INFO] Applying infrastructure...
terraform apply

pause