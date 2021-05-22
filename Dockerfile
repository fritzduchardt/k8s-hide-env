FROM golang:1.16.2 as builder
WORKDIR /go/src/github.com/fritzduchardt/k8shideenv/
COPY cmd/k8s-hide-env/ /go/src/github.com/fritzduchardt/k8shideenv/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o app *.go
RUN go get -d -v

FROM alpine:latest
COPY --from=builder /go/src/github.com/fritzduchardt/k8shideenv/app .
COPY /cmd/k8s-hide-env/app .
CMD ["./app"]
