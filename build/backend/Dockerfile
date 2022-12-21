FROM golang:latest

WORKDIR /app 

COPY ../ .

RUN go mod tidy

RUN GOOS=darwin GOARCH=arm64 go build -o  password-manager ./cmd/main/main.go

CMD ["cat", "password-manager"]
