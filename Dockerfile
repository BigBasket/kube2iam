FROM golang:1.16.0 AS builder

WORKDIR /app

ENV GO111MODULE=on \
  CGO_ENABLED=1 \
  GOOS=linux \
  GOARCH=amd64

COPY . ./

RUN go build -o /app/main /app/cmd
	
FROM ubuntu:20.04

COPY --from=BUILDER /etc/ssl/certs/ /etc/ssl/certs/
RUN apt-get update && apt-get install -y iptables 

COPY --from=builder /app/main .

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /
ENV TZ=Asia/Kolkata
ENV ZONEINFO=/zoneinfo.zip

CMD ["./main"]
