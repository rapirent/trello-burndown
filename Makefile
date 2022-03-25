install:
	go install .

build:
	GOOS=darwin GOARCH=arm64 go build -o build/trello-burndown_${GOOS}_${GOARCH} ./cmd/

docker:
	docker build --no-cache -t trello-burndown .

run: install
	trello-burndown

.PNONY: install build docker run
