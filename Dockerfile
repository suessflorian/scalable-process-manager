FROM golang:1.17

WORKDIR /go/src/build

COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN GOOS=linux GOARCH=amd64 go build -o process-manager

CMD ["/go/src/build/process-manager"]
