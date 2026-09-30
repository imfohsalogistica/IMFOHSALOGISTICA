@echo off
chcp 65001 >nul
taskkill /IM server_imfohsa.exe /F >nul 2>&1
taskkill /IM ngrok.exe /F >nul 2>&1
taskkill /IM cloudflared.exe /F >nul 2>&1
echo Piloto IMFOHSA detenido.
timeout /t 2 /nobreak >nul
