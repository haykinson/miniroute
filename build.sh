GOARCH=arm GOOS=linux go build -o bin/parse-linux-arm parse.go service.pb.go logging.go
GOARCH=amd64 GOOS=linux go build -o bin/serve-linux serve.go service.pb.go logging.go
