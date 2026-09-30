@echo off
chcp 65001 >nul
cls
echo Verificando LOGISTICA IMFOHSA R7.5...
powershell -NoProfile -Command "try { $r=Invoke-RestMethod -TimeoutSec 4 http://localhost:8815/api/health; Write-Host 'SERVIDOR ACTIVO - PUERTO 8815' -ForegroundColor Green; Write-Host ('Version datos: ' + $r.version) } catch { Write-Host 'SERVIDOR SIN RESPUESTA. Ejecute INICIAR_IMFOHSA.bat.' -ForegroundColor Red }"
pause
