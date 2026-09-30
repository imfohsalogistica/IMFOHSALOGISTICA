@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"
title DIAGNOSTICO LOGISTICA IMFOHSA
cls
echo ===============================================
echo DIAGNOSTICO DE ARRANQUE IMFOHSA R7.7.5
echo ===============================================
echo.
echo Carpeta: %CD%
echo.
echo [Archivos]
if exist server_imfohsa.exe (echo OK server_imfohsa.exe) else (echo FALTA server_imfohsa.exe)
if exist data\state.json (echo OK data\state.json) else (echo FALTA data\state.json)
echo.
echo [Puerto 8815]
netstat -ano | findstr ":8815"
echo.
echo [Ultimo error]
if exist server_error.log (type server_error.log) else (echo No existe server_error.log)
echo.
echo [Ultimo registro]
if exist server_runtime.log (type server_runtime.log) else (echo No existe server_runtime.log)
echo.
echo Presione cualquier tecla para cerrar el diagnostico.
pause >nul
