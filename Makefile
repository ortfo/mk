build:
	go mod tidy
	go build

install:
	chmod +x mk
	sudo cp mk ~/.local/bin/ortfomk
