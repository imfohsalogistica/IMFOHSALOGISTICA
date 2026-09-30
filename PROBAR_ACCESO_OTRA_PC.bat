@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
title IMFOHSA - Diagnostico de red
cls

echo ============================================================
echo  DIAGNOSTICO PARA PRUEBAS DESDE OTRA COMPUTADORA
echo ============================================================
echo.

powershell -NoProfile -ExecutionPolicy Bypass -Command "try { $i=Invoke-RestMethod -Uri 'http://localhost:8815/api/info' -TimeoutSec 5; Write-Host ('Servidor: OK'); Write-Host ('Red local: ' + $i.lanUrl); Write-Host ('HTTPS publico: ' + $i.publicUrl); } catch { Write-Host 'Servidor: NO RESPONDE'; exit 1 }"
if errorlevel 1 (
  echo.
  echo Ejecute primero INICIAR_IMFOHSA.bat.
  pause
  exit /b 1
)

echo.
echo PASOS EN LA OTRA COMPUTADORA:
echo 1. Abra ENLACE_RED_LOCAL.txt y escriba exactamente esa direccion en Chrome.
echo 2. Ambas computadoras deben estar en la misma red/Wi-Fi.
echo 3. Si no abre, revise que la red de Windows de la PC principal este marcada como PRIVADA.
echo 4. Si Windows Firewall muestra una ventana para server_imfohsa.exe, permita REDES PRIVADAS.
echo 5. Para celular, camara y GPS, utilice el enlace HTTPS cuando este disponible.
echo.
echo Este diagnostico NO cambia Firewall ni antivirus.
echo.
pause
