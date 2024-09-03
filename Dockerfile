FROM golang:1.22 AS builder
WORKDIR /workspace
COPY . .
RUN go build -buildvcs=false -o app .

FROM debian:latest
RUN \
 apt-get update && \
 apt-get install ca-certificates curl vim -y
WORKDIR /workspace
COPY --from=builder /workspace/app .
COPY --from=builder /workspace/config.yaml .
COPY --from=builder /workspace/config-atlasnet.yaml .
COPY --from=builder /workspace/id.json .

CMD ["./app"]
