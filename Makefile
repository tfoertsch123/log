test:
	go test -v -vet=off -count=1 -coverprofile=cover.out

cover: test
	go tool cover -html=cover.out -o coverage.html

doc:
	godoc -http=127.0.0.1:6060
