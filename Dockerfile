FROM golang:1.14.0-alpine
WORKDIR /data
RUN apk add --no-cache git
COPY go.mod ./
COPY cmd cmd
COPY internal internal
ARG LDFlags
RUN go build -ldflags "$LDFlags" ./cmd/server.go

FROM  alpine:3.9.5
WORKDIR /prom_adaptor
#COPY conf conf
COPY --from=0 /data/server /prom_adaptor
ENTRYPOINT ["./server"]
