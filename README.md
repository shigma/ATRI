# atri

アトリは、高性能ですから!

```bat
REM Step 0: init vs env
%comspec% /k "C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\VsDevCmd.bat"

REM "Step 1: cgo"
go build -buildmode c-shared -o request.dll request.go

REM "Step 2: patch cgo header"
REM "Comment out invalid lines in request.h"

REM "Step 3: generate lib from dll"
dumpbin /exports request.dll > exports.txt
echo LIBRARY request > request.def
echo EXPORTS >> request.def
for /f "skip=19 tokens=4" %%A in (exports.txt) do echo %%A >> request.def
lib /def:request.def /out:request.lib /machine:X64

REM "Step 4: node-gyp"
node-gyp configure
node-gyp build

REM "Step 5: copy dependency"
copy request.dll build\Release

REM "Step 6: test it out"
node ./test.js
```
