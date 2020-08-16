cd ./src
go build -buildmode c-archive -o ./main.a
cd ../bind
node-gyp rebuild
