@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
title IMFOHSA - VERIFICACION HTTPS R7.5.13
cls
echo ============================================================
echo  PRUEBA DE ENLACE PUBLICO - LOGISTICA IMFOHSA R7.5.13
echo ============================================================
echo.
echo 1. Verificando servidor local...
powershell -NoProfile -Command "try {$r=Invoke-RestMethod -TimeoutSec 5 http://localhost:8815/api/health;if($r.ok){exit 0}else{exit 1}}catch{exit 1}"
if errorlevel 1 (
  echo [ERROR] El servidor local no esta activo. Ejecute INICIAR_IMFOHSA.bat primero.
  pause
  exit /b 1
)
echo [OK] Servidor local activo.
echo.
echo 2. Esperando enlace HTTPS verificado...
set COUNT=0
:WAIT_LINK
set /a COUNT+=1
for /f "usebackq tokens=*" %%L in (`powershell -NoProfile -Command "$p='ENLACES_DE_ACCESO.txt'; if(Test-Path $p){$m=Select-String -Path $p -Pattern 'https://[A-Za-z0-9-]+\.trycloudflare\.com' | Select-Object -First 1; if($m){[regex]::Match($m.Line,'https://[A-Za-z0-9-]+\.trycloudflare\.com').Value}}"`) do set PUBLIC=%%L
if not defined PUBLIC (
  if !COUNT! GEQ 90 goto NO_LINK
  echo Preparando HTTPS... !COUNT!/90
  timeout /t 1 /nobreak >nul
  goto WAIT_LINK
)
echo Encontrado: !PUBLIC!
echo.
echo 3. Comprobando acceso externo real...
powershell -NoProfile -Command "try {$r=Invoke-WebRequest -UseBasicParsing -TimeoutSec 15 '!PUBLIC!/api/health?qa=1'; if($r.StatusCode -ge 200 -and $r.StatusCode -lt 300){Write-Host '[OK] HTTPS PUBLICO FUNCIONA' -ForegroundColor Green; exit 0}else{exit 1}}catch{Write-Host $_.Exception.Message -ForegroundColor Red; exit 1}"
if errorlevel 1 goto BAD_LINK
echo.
echo Puede probar ahora los botones Compartir ruta, Solicitud de vehiculo, Flotilla/fotos y Consulta vendedores.
pause
exit /b 0
:NO_LINK
echo [ERROR] No se genero HTTPS en 90 segundos.
echo Revise conexion a Internet o bloqueo corporativo hacia Cloudflare/GitHub.
pause
exit /b 2
:BAD_LINK
echo [ERROR] Se genero una URL pero no respondio desde Internet.
echo La aplicacion bloqueara Copiar/WhatsApp para evitar enviar un enlace roto.
pause
exit /b 3
