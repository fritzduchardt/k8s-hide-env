FROM golang:1.16.2
WORKDIR /go/src/github.com/fritzduchardt/k8shideenv/
RUN go get -d -v gopkg.in/yaml.v2
COPY src/main/go/k8shideenv  .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
COPY --from=0 /go/src/github.com/fritzduchardt/k8shideenv/app .
CMD ["./app"]  
