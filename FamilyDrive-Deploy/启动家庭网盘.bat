@echo off
chcp 65001
echo ===============================
echo   家庭网盘一键启动器
echo ===============================
echo.

echo 启动后端服务器...
cd backend
start "家庭网盘后端" family-drive-server.exe

echo 等待服务器启动...
timeout /t 3 > nul

echo 启动桌面应用...
cd ..
family-drive.exe

echo.
echo 使用说明：
echo 1. 关闭桌面窗口即可停止使用
echo 2. 如需完全停止，请关闭后端窗口
echo 3. 局域网访问: http://192.168.56.1:8000
echo.
pause
