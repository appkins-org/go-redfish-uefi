FROM ubuntu:latest AS firmware

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV CROSS_COMPILE=aarch64-linux-gnu-
ENV TARGET_BOARD=rpi_4

# Install dependencies
RUN apt update && apt install -y \
    gcc-aarch64-linux-gnu \
    make \
    git \
    bc \
    bison \
    flex \
    libssl-dev \
    libgnutls28-dev \
    device-tree-compiler \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Clone U-Boot repository
RUN git clone --depth=1 -b master https://source.denx.de/u-boot/u-boot.git

# Change into U-Boot directory
WORKDIR /build/u-boot

COPY .config /build/u-boot/.config

# Build U-Boot
RUN make distclean && \
    make ${TARGET_BOARD}_defconfig && \
    make -j$(nproc) CROSS_COMPILE=${CROSS_COMPILE}

# Builder
# Build the Go binary
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
COPY --from=firmware /build/u-boot/u-boot.bin /build/u-boot/arch/arm/dts/bcm2711-rpi-4-b.dtb /build/u-boot/dts/dt.dtb /app/internal/internal/uboot/
RUN CGO_ENABLED=0 GOOS=linux go build -o ./go-redfish-uefi


FROM scratch
COPY --from=builder /app/go-redfish-uefi /go-redfish-uefi
ENTRYPOINT ["/go-redfish-uefi"]
