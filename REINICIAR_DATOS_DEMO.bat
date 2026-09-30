@echo off
chcp 65001 >nul
cd /d "%~dp0"
taskkill /IM server_imfohsa.exe /F >nul 2>&1
taskkill /IM cloudflared.exe /F >nul 2>&1
copy /Y "data\default_state.json" "data\state.json" >nul
if exist ENLACES_DE_ACCESO.txt del /q ENLACES_DE_ACCESO.txt >nul 2>&1
echo.
echo Escenario QA restaurado: 100 clientes y 100 SO SIN ASIGNAR.
echo Todos los pedidos quedan Pendiente para probar Planificador y Asignacion inteligente.
echo Los usuarios y roles se conservan.
echo.
pause
