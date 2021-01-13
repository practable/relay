#/bin/sh
cd ../cmd/book
go build
sudo cp book /usr/local/bin
cd ../shell
go build
sudo cp shell /usr/local/bin
cd ../session
go build
sudo cp session /usr/local/bin

