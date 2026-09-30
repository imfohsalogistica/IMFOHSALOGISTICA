@echo off
setlocal EnableExtensions EnableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
title LOGISTICA IMFOHSA R7.7.7 PILOTO - INTERNET NGROK
cls

echo ============================================================
echo  LOGISTICA IMFOHSA R7.7.7 PILOTO - ACCESO EXTERNO NGROK
echo ============================================================
echo.
echo Esta PC sera el servidor central del piloto.
echo Gerencia, Logistica y mensajeros podran entrar desde OTRAS REDES.
echo Mantenga esta ventana ABIERTA durante toda la prueba.
echo.

if not exist "server_imfohsa.exe" goto MISSING_SERVER
if not exist "data\state.json" goto MISSING_DATA

powershell -NoProfile -ExecutionPolicy Bypass -Command "Unblock-File -LiteralPath '%CD%\server_imfohsa.exe' -ErrorAction SilentlyContinue; if(Test-Path '%CD%\ngrok.exe'){Unblock-File -LiteralPath '%CD%\ngrok.exe' -ErrorAction SilentlyContinue}" >nul 2>&1

taskkill /IM server_imfohsa.exe /F >nul 2>&1
taskkill /IM cloudflared.exe /F >nul 2>&1
taskkill /IM ngrok.exe /F >nul 2>&1

if exist server_runtime.log ren server_runtime.log server_runtime_anterior.log >nul 2>&1
if exist ngrok_runtime.log ren ngrok_runtime.log ngrok_runtime_anterior.log >nul 2>&1
if exist server_error.log del /q server_error.log >nul 2>&1
if exist ngrok_error.log del /q ngrok_error.log >nul 2>&1
if exist DIAGNOSTICO_ARRANQUE.txt del /q DIAGNOSTICO_ARRANQUE.txt >nul 2>&1
if exist ENLACE_PILOTO_PUBLICO.txt del /q ENLACE_PILOTO_PUBLICO.txt >nul 2>&1
if exist ENLACES_PILOTO_PARA_COMPARTIR.txt del /q ENLACES_PILOTO_PARA_COMPARTIR.txt >nul 2>&1

set "NGROKCMD="
if exist "%CD%\ngrok.exe" set "NGROKCMD=%CD%\ngrok.exe"
if not defined NGROKCMD for /f "delims=" %%G in ('where ngrok.exe 2^>nul') do if not defined NGROKCMD set "NGROKCMD=%%G"
if not defined NGROKCMD if exist "%LOCALAPPDATA%\Microsoft\WinGet\Links\ngrok.exe" set "NGROKCMD=%LOCALAPPDATA%\Microsoft\WinGet\Links\ngrok.exe"
if not defined NGROKCMD goto NEED_NGROK

echo [1/5] Verificando puerto 8815...
set "PORTPID="
for /f "tokens=5" %%P in ('netstat -ano ^| findstr /R /C:":8815 .*LISTENING"') do set "PORTPID=%%P"
if defined PORTPID goto PORT_BUSY
echo      Puerto disponible.

echo [2/5] Iniciando servidor IMFOHSA...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$p=Start-Process -FilePath '%CD%\server_imfohsa.exe' -WorkingDirectory '%CD%' -RedirectStandardOutput '%CD%\server_runtime.log' -RedirectStandardError '%CD%\server_error.log' -PassThru; $p.Id | Out-File -Encoding ascii '%CD%\server_pid.txt'" >nul 2>&1

set /a WAIT=0
:WAIT_HEALTH
powershell -NoProfile -Command "try{$r=Invoke-RestMethod -Uri 'http://127.0.0.1:8815/api/health' -TimeoutSec 3;if($r.ok){exit 0}else{exit 1}}catch{exit 1}" >nul 2>&1
if not errorlevel 1 goto SERVER_OK
set /a WAIT+=1
if !WAIT! GEQ 20 goto FAIL_SERVER
timeout /t 1 /nobreak >nul
goto WAIT_HEALTH

:SERVER_OK
echo      Servidor: OK
set "LANURL="
for /f "usebackq delims=" %%L in (`powershell -NoProfile -Command "try{$i=Invoke-RestMethod -Uri 'http://127.0.0.1:8815/api/info' -TimeoutSec 3;if($i.lanUrl){[Console]::Write($i.lanUrl)}}catch{}"`) do set "LANURL=%%L"
if defined LANURL (
  >ENLACE_RED_LOCAL.txt echo !LANURL!/index.html?v=777
  echo      Misma Wi-Fi: !LANURL!/index.html?v=777
)

echo [3/5] Abriendo plataforma local...
start "" "http://localhost:8815/index.html?v=777"

echo [4/5] Iniciando tunel seguro de ngrok...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$p=Start-Process -FilePath '%NGROKCMD%' -ArgumentList @('http','8815','--log=stdout','--log-format=logfmt') -WorkingDirectory '%CD%' -RedirectStandardOutput '%CD%\ngrok_runtime.log' -RedirectStandardError '%CD%\ngrok_error.log' -PassThru; $p.Id | Out-File -Encoding ascii '%CD%\ngrok_pid.txt'" >nul 2>&1

set /a NW=0
:WAIT_NGROK
set "PUBLICURL="
for /f "usebackq delims=" %%U in (`powershell -NoProfile -Command "try{$j=Invoke-RestMethod -Uri 'http://127.0.0.1:4040/api/tunnels' -TimeoutSec 3;$u=$j.tunnels|Where-Object {$_.public_url -like 'https://*'}|Select-Object -First 1 -ExpandProperty public_url;if($u){[Console]::Write($u)}}catch{}"`) do set "PUBLICURL=%%U"
if defined PUBLICURL goto VALIDATE_PUBLIC
set /a NW+=2
if !NW! GEQ 40 goto NGROK_FAIL
timeout /t 2 /nobreak >nul
goto WAIT_NGROK

:VALIDATE_PUBLIC
echo [5/5] Verificando acceso PUBLICO real...
powershell -NoProfile -Command "try{$r=Invoke-WebRequest -Uri '!PUBLICURL!/api/health' -TimeoutSec 15 -UseBasicParsing;if($r.StatusCode -ge 200 -and $r.StatusCode -lt 500){exit 0}else{exit 1}}catch{exit 1}" >nul 2>&1
if errorlevel 1 goto PUBLIC_FAIL

>ENLACE_PILOTO_PUBLICO.txt echo !PUBLICURL!/index.html?v=777
>ENLACES_PILOTO_PARA_COMPARTIR.txt (
 echo LOGISTICA IMFOHSA R7.7.7 PILOTO
 echo =================================
 echo.
 echo ENLACE PUBLICO UNICO PARA TODOS:
 echo !PUBLICURL!/index.html?v=777
 echo.
 echo IMPORTANTE: En ngrok gratuito, la primera visita desde un navegador puede mostrar
 echo una pantalla informativa de ngrok. Presione Visit Site una sola vez.
 echo.
 echo JLEMUS - Gerencia
 echo WFUENTES - Jefe de Logistica
 echo NLIMA - Asistente Zona 16
 echo GOLIVA - Coordinador Zona 4
 echo VENTAS1 - Ventas
 echo COBROS1 - Creditos y Cobros
 echo DCHOC - Daniel Choc
 echo ECIVIL - Edwin Civil
 echo.
 echo Clave temporal: Imfohsa1234/
)

echo.
echo ============================================================
echo  PILOTO EXTERNO LISTO
echo ============================================================
echo Enlace PUBLICO probado:
echo !PUBLICURL!/index.html?v=777
echo.
echo Puede abrirse desde datos moviles u otra red Wi-Fi.
echo NO cierre esta ventana.
start "" notepad.exe "%CD%\ENLACES_PILOTO_PARA_COMPARTIR.txt"
goto MONITOR

:MONITOR
timeout /t 10 /nobreak >nul
powershell -NoProfile -Command "try{$r=Invoke-RestMethod -Uri 'http://127.0.0.1:8815/api/health' -TimeoutSec 3;if($r.ok){exit 0}else{exit 1}}catch{exit 1}" >nul 2>&1
if errorlevel 1 goto FAIL_SERVER_RUNTIME
powershell -NoProfile -Command "try{$j=Invoke-RestMethod -Uri 'http://127.0.0.1:4040/api/tunnels' -TimeoutSec 3;if($j.tunnels.Count -gt 0){exit 0}else{exit 1}}catch{exit 1}" >nul 2>&1
if errorlevel 1 goto NGROK_DROPPED
goto MONITOR

:NEED_NGROK
cls
echo ============================================================
echo  FALTA CONFIGURAR NGROK
echo ============================================================
echo.
echo Para probar con un mensajero en otra red necesitamos un tunel externo.
echo Cloudflare fue bloqueado/inestable en esta red; R7.7.7 usa ngrok.
echo.
echo 1. Ejecute CONFIGURAR_NGROK.bat
 echo 2. Cree/inicie una cuenta gratuita en ngrok
 echo 3. Pegue su AUTHTOKEN solamente en esa ventana
 echo 4. Vuelva a ejecutar INICIAR_PILOTO_INTERNET.bat
 echo.
start "" "%CD%\CONFIGURAR_NGROK.bat"
pause
exit /b 1

:NGROK_FAIL
cls
echo ============================================================
echo  NGROK NO PUDO INICIAR
 echo ============================================================
echo.
echo El servidor IMFOHSA SI esta funcionando localmente.
echo.
if exist ngrok_error.log type ngrok_error.log
if exist ngrok_runtime.log type ngrok_runtime.log
>DIAGNOSTICO_ARRANQUE.txt echo ERROR: ngrok no genero endpoint HTTPS.
>>DIAGNOSTICO_ARRANQUE.txt echo Revise ngrok_runtime.log y ngrok_error.log.
echo.
echo Si aparece ERR_NGROK_4018 o autenticacion, ejecute CONFIGURAR_NGROK.bat.
pause
exit /b 1

:PUBLIC_FAIL
cls
echo ============================================================
echo  NGROK GENERO URL PERO NO RESPONDIO
 echo ============================================================
echo.
echo No comparta el enlace hasta resolverlo.
if exist ngrok_runtime.log type ngrok_runtime.log
pause
exit /b 1

:NGROK_DROPPED
echo.
echo ADVERTENCIA: se perdio el tunel de ngrok.
echo Los usuarios externos dejaran de poder entrar hasta reiniciar el piloto.
echo Revise ngrok_runtime.log.
pause
exit /b 1

:FAIL_SERVER
cls
echo ============================================================
echo  NO SE PUDO INICIAR EL SERVIDOR
 echo ============================================================
if exist server_error.log type server_error.log
if exist server_runtime.log type server_runtime.log
pause
exit /b 1

:FAIL_SERVER_RUNTIME
echo.
echo ERROR: el servidor IMFOHSA se detuvo durante la prueba.
if exist server_error.log type server_error.log
pause
exit /b 1

:PORT_BUSY
echo.
echo Otro programa esta usando el puerto 8815.
echo Cierre cualquier version anterior de IMFOHSA o use DETENER_PILOTO.bat.
pause
exit /b 1

:MISSING_SERVER
echo ERROR: falta server_imfohsa.exe. Descomprima el ZIP completo.
pause
exit /b 1

:MISSING_DATA
echo ERROR: falta data\state.json. Descomprima el ZIP completo.
pause
exit /b 1
