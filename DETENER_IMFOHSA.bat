@echo off
taskkill /IM server_imfohsa.exe /F >nul 2>&1
taskkill /IM cloudflared.exe /F >nul 2>&1
echo IMFOHSA detenido.
timeout /t 2 >nul
