@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
title LOGISTICA IMFOHSA R7.7.4 PILOTO - PLANIFICADOR INTELIGENTE + INVENTARIO
cls
echo ============================================================
echo  LOGISTICA IMFOHSA R7.7.4 PILOTO - PLANIFICADOR INTELIGENTE + INVENTARIO
echo ============================================================
echo.
echo Panel local: http://localhost:8815/index.html
echo Los enlaces publicos HTTPS se preparan en segundo plano.
echo No se desactiva antivirus ni se crean excepciones de seguridad.
echo Mantenga esta ventana abierta mientras use esta version de pruebas.
echo.

taskkill /IM server_imfohsa.exe /F >nul 2>&1
taskkill /IM cloudflared.exe /F >nul 2>&1
if exist ENLACES_DE_ACCESO.txt del /q ENLACES_DE_ACCESO.txt >nul 2>&1

set FIRST=1
:START_SERVER
set FAILS=0
echo [%date% %time%] Iniciando servidor R7.7.4 PILOTO...
start "" /B server_imfohsa.exe >> server_runtime.log 2>&1

:WAIT_HEALTH
powershell -NoProfile -Command "try { $r=Invoke-RestMethod -TimeoutSec 5 http://localhost:8815/api/health; if($r.ok){exit 0}else{exit 1} } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  timeout /t 1 /nobreak >nul
  tasklist /FI "IMAGENAME eq server_imfohsa.exe" 2>nul | find /I "server_imfohsa.exe" >nul || goto START_SERVER
  goto WAIT_HEALTH
)

if "%FIRST%"=="1" (
  set FIRST=0
  echo Servidor ACTIVO en puerto 8815.
  echo.
  for /f "usebackq delims=" %%I in (`powershell -NoProfile -ExecutionPolicy Bypass -Command "$i=Invoke-RestMethod -Uri 'http://localhost:8815/api/info' -TimeoutSec 5; if($i.lanUrl){[Console]::Write($i.lanUrl)}"`) do set "LANURL=%%I"
  if defined LANURL (
    echo Enlace otra PC: !LANURL!/index.html?v=774
    ^>ENLACE_RED_LOCAL.txt echo !LANURL!/index.html?v=774
  ) else (
    echo Enlace otra PC: no detectado. Use MOSTRAR_ENLACE_RED_LOCAL.bat
  )
  echo.
  echo Preparando HTTPS publico. La aplicacion NO comparte enlaces hasta verificarlos.
  start "" "http://localhost:8815/index.html?v=774"
)

:MONITOR
timeout /t 10 /nobreak >nul
powershell -NoProfile -Command "try { $r=Invoke-RestMethod -TimeoutSec 4 http://localhost:8815/api/health; if($r.ok){exit 0}else{exit 1} } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  set /a FAILS+=1
  echo [%date% %time%] Aviso: servidor local sin respuesta !FAILS!/6.
  if !FAILS! GEQ 6 goto RECOVER
) else (
  set FAILS=0
)
goto MONITOR

:RECOVER
echo [%date% %time%] Seis comprobaciones consecutivas fallaron. Reiniciando servidor...
taskkill /IM server_imfohsa.exe /F >nul 2>&1
taskkill /IM cloudflared.exe /F >nul 2>&1
timeout /t 1 /nobreak >nul
goto START_SERVER
