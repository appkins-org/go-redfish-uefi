FROM golang:1.23.5 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /go-redfish-uefi


FROM scratch
COPY --from=builder /go-redfish-uefi /go-redfish-uefi
ENTRYPOINT ["/go-redfish-uefi"]
