FROM golang:1.14.0-alpine
WORKDIR /data
RUN apk add --no-cache git
COPY go.mod ./
COPY cmd cmd
COPY internal internal
RUN go build cmd/server.go

FROM  alpine:3.9.5
WORKDIR /prom_adptor
#COPY conf conf
COPY --from=0 /data/server /prom_adptor
ENTRYPOINT ["./server"]
