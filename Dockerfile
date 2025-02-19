FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./go-redfish-uefi


FROM scratch
COPY --from=builder /app/go-redfish-uefi /go-redfish-uefi
ENTRYPOINT ["/go-redfish-uefi"]
