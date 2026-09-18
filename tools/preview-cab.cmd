@echo off
rem Preview the launch animation on this PC with a Blockbench export.
rem Export as astrocade-exp1.obj (+ .mtl, texture.png) into the repo's
rem meshes\ folder, then double-click this file. It copies the model into
rem the app, records the animation with real screenshots, writes
rem launch-cab.gif and sheet.png into meshes\ and opens the GIF.
setlocal
set ROOT=%~dp0..
set MESH=%ROOT%\..\..\..\meshes
if not exist "%MESH%" set MESH=%ROOT%\meshes
set GO=go.exe
where go.exe >nul 2>nul || set GO=C:\Users\user\AppData\Local\Programs\Go\bin\go.exe
set IMG=C:\Users\user\Dropbox\Claude\misterzine\docs\images
cd /d "%ROOT%" || exit /b 1
copy /y "%MESH%\astrocade-exp1.obj" internal\app\cab.obj >nul || exit /b 1
copy /y "%MESH%\texture.png" internal\app\cab-texture.png >nul || exit /b 1
powershell -NoProfile -Command "(Get-Content internal\app\cab.obj) -replace '^mtllib .*','mtllib cab.mtl' | Set-Content -Encoding ascii internal\app\cab.obj; (Get-Content '%MESH%\astrocade-exp1.mtl') -replace '^map_Kd .*','map_Kd cab-texture.png' | Set-Content -Encoding ascii internal\app\cab.mtl"
if exist out\cab rmdir /s /q out\cab
mkdir out\cab
"%GO%" run ./cmd/mzharness -motion -images "%IMG%" -out out\cab -script "shot list; enter; wait 1500; start; frames cab 33 60" >nul || exit /b 1
python "%~dp0preview-cab.py" out\cab "%MESH%"
start "" "%MESH%\launch-cab.gif"
