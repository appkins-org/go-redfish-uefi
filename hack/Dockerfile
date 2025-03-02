# Use a Debian-based image with necessary tools
FROM ubuntu:latest

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV CROSS_COMPILE=aarch64-linux-gnu-
ENV TARGET_BOARD=rpi_4
ENV OUTPUT_FILE=armstub8.bin

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

# Build U-Boot
RUN make distclean && \
    make ${TARGET_BOARD}_defconfig
   
COPY .config /build/u-boot/
    
RUN make -j$(nproc) CROSS_COMPILE=${CROSS_COMPILE}

# Create output directory
RUN mkdir -p /firmware

# Copy the correct binary (u-boot.bin, which is needed for armstub)
RUN cp u-boot.bin /firmware/${OUTPUT_FILE}

# Set the output directory as a volume
VOLUME [ "/firmware" ]

# Define the entrypoint to copy the output when running the container
CMD cp /firmware/${OUTPUT_FILE} /firmware