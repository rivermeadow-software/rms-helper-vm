# create custom alpine linux iso using aports
FROM alpine:3.22
RUN apk update && apk upgrade
RUN apk add --no-cache bash curl xorriso syslinux mtools abuild alpine-conf squashfs-tools grub git doas openssh coreutils vim qemu-guest-agent sysklogd-openrc
RUN git clone --depth=1 https://gitlab.alpinelinux.org/alpine/aports.git
RUN abuild-keygen -a -n
RUN chmod +x aports/scripts/mkimage.sh

COPY mkimg.helper.sh aports/scripts
RUN mkdir /iso

#RUN addgroup root abuild

# Set TMPDIR if /tmp is too small (e.g., in a default Docker container)
# Adjust the path if you prefer a different location
ENV TMPDIR=/root/tmp
RUN mkdir -pv ${TMPDIR}

# Use custom base profile with virt kernel and minimal packages
COPY mkimg.base.sh aports/scripts/mkimg.base.sh
RUN chmod +x aports/scripts/mkimg.base.sh

# Use standard profile
COPY mkimg.standard.sh aports/scripts/mkimg.standard.sh
RUN chmod +x aports/scripts/mkimg.standard.sh

COPY genapkovl-mkimgoverlay.sh aports/scripts/genapkovl-mkimgoverlay.sh
RUN chmod +x aports/scripts/genapkovl-mkimgoverlay.sh
COPY traffic-filter.conf aports/scripts/traffic-filter.conf

#RUN abuild-keygen -i -a -n

COPY rmstool aports/scripts/rmstool

#WORKDIR /build

CMD ["sh", "aports/scripts/mkimage.sh", "--tag", "v3.22", "--arch", "x86_64", "--outdir", "/iso", "--repository", "http://10.0.0.85/v3.22/main", "--repository","http://10.0.0.85/v3.22/community", "--profile", "helper_kvm"]