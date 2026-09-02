@echo off
rem 飞光 Lumifly 前端启动脚本
cd /d %~dp0
if not exist node_modules (
  echo 正在安装依赖...
  call npm install
)
echo 前端已启动：http://localhost:5176
call npm run dev
