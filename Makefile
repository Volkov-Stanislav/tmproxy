build:
	go build -o cmd/tmproxy tmproxy.go

run:
	go run tmproxy.go

docker_build: 
	docker build -t tmproxy .

docker_run: 
	docker run -i -t --rm -p=8080:8080 --name="tmproxy" tmproxy