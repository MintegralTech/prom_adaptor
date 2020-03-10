FROM golang:1.13.8-alpine
WORKDIR /data
COPY go.mod server.go ./
RUN go build server.go
FROM  alpine:3.9.5
WORKDIR /prome_adpter
COPY --from=0 /data/server /prome_adpter
ENTRYPOINT ["./server"]



