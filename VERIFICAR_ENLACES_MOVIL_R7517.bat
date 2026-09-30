@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
title IMFOHSA R7.5.17 - Verificar enlaces movil
cls

echo ============================================================
echo  LOGISTICA IMFOHSA R7.5.17 - PRUEBA DE ENLACES
echo ============================================================
echo.
echo 1. Verificando servidor local...
powershell -NoProfile -ExecutionPolicy Bypass -Command "try{$h=Invoke-RestMethod 'http://localhost:8815/api/health' -TimeoutSec 5;if($h.ok){exit 0}else{exit 1}}catch{exit 1}" >nul 2>&1
if errorlevel 1 (
  echo [ERROR] No responde el servidor. Ejecute INICIAR_IMFOHSA.bat primero.
  pause
  exit /b 1
)
echo [OK] Servidor local activo.

echo.
echo 2. Detectando enlace LAN...
set "LANURL="
for /f "usebackq delims=" %%I in (`powershell -NoProfile -ExecutionPolicy Bypass -Command "$i=Invoke-RestMethod 'http://localhost:8815/api/info' -TimeoutSec 5; if($i.lanUrl){[Console]::Write($i.lanUrl)}"`) do set "LANURL=%%I"
if defined LANURL (
  echo [OK] Misma Wi-Fi: !LANURL!/index.html?v=7517
) else (
  echo [AVISO] No se detecto IP local.
)

echo.
echo 3. Solicitando / esperando HTTPS publico...
powershell -NoProfile -ExecutionPolicy Bypass -Command "try{Invoke-WebRequest -Method Post -Uri 'http://localhost:8815/api/tunnel/restart' -UseBasicParsing -TimeoutSec 5 | Out-Null}catch{}" >nul 2>&1
set "PUBLICURL="
for /L %%N in (1,1,70) do (
  for /f "usebackq delims=" %%I in (`powershell -NoProfile -ExecutionPolicy Bypass -Command "try{$i=Invoke-RestMethod 'http://localhost:8815/api/info' -TimeoutSec 4;if($i.publicUrl){[Console]::Write($i.publicUrl)}}catch{}"`) do set "PUBLICURL=%%I"
  if defined PUBLICURL goto CHECK_PUBLIC
  timeout /t 1 /nobreak >nul
)

echo [AVISO] No se obtuvo HTTPS dentro del tiempo de prueba.
echo El enlace LAN puede seguir utilizandose dentro de la misma Wi-Fi.
goto WRITE

:CHECK_PUBLIC
echo Candidato HTTPS: !PUBLICURL!
echo Verificando desde Internet...
powershell -NoProfile -ExecutionPolicy Bypass -Command "try{$r=Invoke-WebRequest -Uri '!PUBLICURL!/api/health?ts=' + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() -UseBasicParsing -TimeoutSec 10;if($r.StatusCode -ge 200 -and $r.StatusCode -lt 400){exit 0}else{exit 1}}catch{exit 1}" >nul 2>&1
if errorlevel 1 (
  echo [AVISO] HTTPS fue generado, pero todavia no responde. Use el boton Reintentar HTTPS dentro de la aplicacion.
) else (
  echo [OK] HTTPS PUBLICO FUNCIONA: !PUBLICURL!/index.html?v=7517
)

:WRITE
>ENLACES_PRUEBA_R7517.txt echo LOGISTICA IMFOHSA R7.5.17
>>ENLACES_PRUEBA_R7517.txt echo.
if defined LANURL >>ENLACES_PRUEBA_R7517.txt echo MISMA WIFI: !LANURL!/index.html?v=7517
if defined PUBLICURL >>ENLACES_PRUEBA_R7517.txt echo CUALQUIER RED: !PUBLICURL!/index.html?v=7517
>>ENLACES_PRUEBA_R7517.txt echo.
>>ENLACES_PRUEBA_R7517.txt echo LAN solo funciona en la misma red. HTTPS funciona desde datos moviles u otra Wi-Fi mientras la PC principal y el tunel permanezcan activos.
echo.
echo Resultado guardado en ENLACES_PRUEBA_R7517.txt
echo.
pause
