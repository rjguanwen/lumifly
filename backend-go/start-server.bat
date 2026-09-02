@echo off
rem 飞光 Lumifly 后端启动脚本（编译并运行）
cd /d %~dp0
set "GOTOOLCHAIN=local"
go build -o lumifly-server.exe ./cmd/server || goto :err
start "lumifly-server" lumifly-server.exe
echo 后端已启动：http://localhost:8004
goto :eof
:err
echo 编译失败，请检查 Go 环境
pause
