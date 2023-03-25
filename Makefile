install_dir = $(HOME)/.local/bin

build: fmt
	go build

fmt:
	go fmt

install:
	cp ./tcprelay $(install_dir)

uninstall:
	rm -f $(install_dir)/tcprelay
