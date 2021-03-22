FROM golang:1.16.2 as builder
WORKDIR /go/src/github.com/fritzduchardt/k8shideenv/
RUN go get -d -v gopkg.in/yaml.v2
COPY src/main/go/k8shideenv  .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o app *.go

FROM alpine:latest
COPY --from=builder /go/src/github.com/fritzduchardt/k8shideenv/app .
CMD ["./app"]  
