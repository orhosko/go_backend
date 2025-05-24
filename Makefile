.PHONY: run, serve, build, clean

run:
	go run main.go

serve:
	~/go/bin/air -c .air.toml

build:
	go build -o main main.go

clean:
	rm main