FROM golang:1.13.8-alpine
WORKDIR /data
COPY go.mod cmd/ internal ./
RUN go build server.go

FROM  alpine:3.9.5
WORKDIR /prom_adpter
COPY conf/ /prom_adpter/
COPY --from=0 /data/server /prom_adpter
ENTRYPOINT ["./server"]



