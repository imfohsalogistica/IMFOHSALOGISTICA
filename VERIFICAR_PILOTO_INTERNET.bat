@echo off
setlocal EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
echo ============================================================
echo VERIFICACION PILOTO INTERNET - IMFOHSA R7.7.1
echo ============================================================
echo.
powershell -NoProfile -Command "try{$h=Invoke-RestMethod -Uri 'http://localhost:8815/api/health' -TimeoutSec 5; Write-Host 'Servidor local: OK'}catch{Write-Host 'Servidor local: NO DISPONIBLE'; exit 1}"
if errorlevel 1 goto END
powershell -NoProfile -Command "$i=Invoke-RestMethod -Uri 'http://localhost:8815/api/info' -TimeoutSec 5; Write-Host ('LAN: '+$i.lanUrl); Write-Host ('HTTPS publico: '+$i.publicUrl); Write-Host ('Tunel activo: '+$i.tunnelRunning)"
if exist ENLACES_DE_ACCESO.txt (
  echo.
  type ENLACES_DE_ACCESO.txt
)
:END
echo.
pause
