all: fwdmng

fwdmng::
	go build

fwdmng.exe::
	GOOS=windows GOARCH=amd64 go build

clean:
	rm -f fwdmng fwdmng.exe
