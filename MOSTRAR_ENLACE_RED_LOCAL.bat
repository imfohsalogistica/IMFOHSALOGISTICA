@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
title IMFOHSA - Enlace para otra computadora
cls

echo ============================================================
echo  LOGISTICA IMFOHSA - ENLACE DE RED LOCAL
echo ============================================================
echo.

echo Verificando que el servidor IMFOHSA este activo...
powershell -NoProfile -ExecutionPolicy Bypass -Command "try { $r=Invoke-RestMethod -Uri 'http://localhost:8815/api/health' -TimeoutSec 5; if($r.ok){exit 0}else{exit 1} } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo [ERROR] El servidor no responde en http://localhost:8815
  echo Primero ejecute INICIAR_IMFOHSA.bat y mantenga esa ventana abierta.
  echo.
  pause
  exit /b 1
)

set "LANURL="
for /f "usebackq delims=" %%I in (`powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $i=Invoke-RestMethod -Uri 'http://localhost:8815/api/info' -TimeoutSec 5; if($i.lanUrl){[Console]::Write($i.lanUrl)}"`) do set "LANURL=%%I"

if not defined LANURL (
  echo [ERROR] El servidor no pudo detectar una direccion IPv4 de red local.
  echo.
  echo Ejecute IPCONFIG y busque la linea Direccion IPv4 de Wi-Fi o Ethernet.
  echo Debe verse parecido a 192.168.1.25 o 10.0.0.25.
  echo.
  pause
  exit /b 1
)

set "FULLURL=!LANURL!/index.html?v=7517"

echo [OK] Servidor local activo.
echo [OK] Direccion de red detectada correctamente.
echo.
echo ============================================================
echo  ENLACE PARA OTRA COMPUTADORA EN LA MISMA RED / WI-FI
echo ============================================================
echo.
echo !FULLURL!
echo.
>ENLACE_RED_LOCAL.txt echo !FULLURL!

echo El enlace tambien quedo guardado en ENLACE_RED_LOCAL.txt
echo.
echo IMPORTANTE:
echo - NO comparta localhost: localhost solo funciona en esta computadora.
echo - La otra computadora debe estar conectada a la misma red/Wi-Fi.
echo - Mantenga INICIAR_IMFOHSA.bat abierto en esta computadora.
echo - Si Windows pregunta por Firewall, permita la aplicacion solo en REDES PRIVADAS.
echo - No se modifica automaticamente el Firewall ni el antivirus.
echo - Camara/GPS desde celular requiere preferiblemente el enlace HTTPS publico.
echo.
echo Probando el enlace de red desde esta computadora...
powershell -NoProfile -ExecutionPolicy Bypass -Command "try { $r=Invoke-RestMethod -Uri '!LANURL!/api/health' -TimeoutSec 5; if($r.ok){Write-Host '[OK] El enlace de red responde correctamente en la computadora principal.'; exit 0}else{exit 1} } catch { Write-Host '[AVISO] No fue posible validar el enlace por IP local.'; exit 1 }"
echo.
pause
