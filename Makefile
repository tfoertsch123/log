test:
	go test -v -count=1 -coverprofile=cover.out ./
	go test -v -count=1 -coverprofile=rotate/cover.out ./rotate/

cover: test
	go tool cover -html=cover.out -o coverage.html
	go tool cover -html=rotate/cover.out -o rotate/coverage.html

doc:
	godoc -http=127.0.0.1:6060
