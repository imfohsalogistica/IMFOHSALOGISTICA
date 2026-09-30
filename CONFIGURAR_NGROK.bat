@echo off
setlocal EnableExtensions
chcp 65001 >nul
cd /d "%~dp0"
title CONFIGURAR NGROK - LOGISTICA IMFOHSA R7.7.7
cls

echo ============================================================
echo  CONFIGURAR NGROK PARA PILOTO EXTERNO IMFOHSA
echo ============================================================
echo.
echo Este paso se realiza UNA SOLA VEZ en la PC servidor.
echo No publique ni comparta su authtoken.
echo.

set "NGROKCMD="
if exist "%CD%\ngrok.exe" set "NGROKCMD=%CD%\ngrok.exe"
if not defined NGROKCMD for /f "delims=" %%G in ('where ngrok.exe 2^>nul') do if not defined NGROKCMD set "NGROKCMD=%%G"
if not defined NGROKCMD if exist "%LOCALAPPDATA%\Microsoft\WinGet\Links\ngrok.exe" set "NGROKCMD=%LOCALAPPDATA%\Microsoft\WinGet\Links\ngrok.exe"

if not defined NGROKCMD (
  echo ngrok no esta instalado.
  echo.
  echo Intentando instalarlo con WinGet...
  where winget >nul 2>&1
  if not errorlevel 1 (
    winget install --id Ngrok.Ngrok -e --source winget --accept-package-agreements --accept-source-agreements
  )
  echo.
  set "NGROKCMD="
  if exist "%CD%\ngrok.exe" set "NGROKCMD=%CD%\ngrok.exe"
  if not defined NGROKCMD for /f "delims=" %%G in ('where ngrok.exe 2^>nul') do if not defined NGROKCMD set "NGROKCMD=%%G"
  if not defined NGROKCMD if exist "%LOCALAPPDATA%\Microsoft\WinGet\Links\ngrok.exe" set "NGROKCMD=%LOCALAPPDATA%\Microsoft\WinGet\Links\ngrok.exe"
)

if not defined NGROKCMD (
  echo.
  echo No fue posible encontrar ngrok automaticamente.
  echo Se abrira la pagina oficial de descarga.
  start "" "https://ngrok.com/download/windows"
  echo.
  echo Instale ngrok, cierre esta ventana y vuelva a ejecutar CONFIGURAR_NGROK.
  pause
  exit /b 1
)

echo ngrok encontrado:
echo %NGROKCMD%
echo.
echo Se abrira la pagina de su AUTHTOKEN en ngrok.
start "" "https://dashboard.ngrok.com/get-started/your-authtoken"
echo.
echo Copie el AUTHTOKEN de su cuenta. Cuando lo tenga, vuelva aqui.
echo El token NO se mostrara mientras lo pega.
echo.

powershell -NoProfile -ExecutionPolicy Bypass -Command "$s=Read-Host 'Pegue su AUTHTOKEN de ngrok' -AsSecureString; $b=[Runtime.InteropServices.Marshal]::SecureStringToBSTR($s); try{$t=[Runtime.InteropServices.Marshal]::PtrToStringBSTR($b); & '%NGROKCMD%' config add-authtoken $t; exit $LASTEXITCODE} finally {[Runtime.InteropServices.Marshal]::ZeroFreeBSTR($b)}"
if errorlevel 1 (
  echo.
  echo No se pudo guardar el authtoken. Revise que lo haya copiado completo.
  pause
  exit /b 1
)

echo.
echo ============================================================
echo  NGROK CONFIGURADO CORRECTAMENTE
echo ============================================================
echo Ahora ejecute INICIAR_PILOTO_INTERNET.
echo.
pause
