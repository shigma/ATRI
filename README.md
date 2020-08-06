# atri

アトリは、高性能ですから!

```bat
REM Step 0: init vs env
%comspec% /k "C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\VsDevCmd.bat"

REM "Step 1: cgo"
go build -buildmode c-shared -o calculate_pi.dll calculate_pi.go

REM "Step 2: patch cgo header"
REM "Comment out invalid lines in calculate_pi.h"

REM "Step 3: generate lib from dll"
dumpbin /exports calculate_pi.dll > exports.txt
echo LIBRARY CALCULATE_PI > calculate_pi.def
echo EXPORTS >> calculate_pi.def
for /f "skip=19 tokens=4" %%A in (exports.txt) do echo %%A >> calculate_pi.def
lib /def:calculate_pi.def /out:calculate_pi.lib /machine:X64

REM "Step 4: node-gyp"
node-gyp configure
node-gyp build

REM "Step 5: copy dependency"
copy calculate_pi.dll build\Release

REM "Step 6: test it out"
PowerShell -Command "echo 'const addon = require(''./build/Release/node-calculator''); console.log(addon.calculate_pi(10000))' | node"
```
