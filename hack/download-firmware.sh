#!/usr/bin/env bash

curl -sfLo internal/firmware/uboot/overlays/rpi-poe-plus.dtbo https://github.com/raspberrypi/firmware/raw/refs/heads/master/boot/overlays/rpi-poe-plus.dtbo
curl -sfLo internal/firmware/uboot/overlays/upstream-pi4.dtbo https://github.com/raspberrypi/firmware/raw/refs/heads/master/boot/overlays/upstream-pi4.dtbo
