build:
	go mod tidy
	cd cmd/ortfomk; go build

install:
	chmod +x cmd/ortfomk/ortfomk
	cp cmd/ortfomk/ortfomk ~/.local/bin/ortfomk
