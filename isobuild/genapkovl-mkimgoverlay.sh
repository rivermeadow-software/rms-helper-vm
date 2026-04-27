#!/bin/sh -e

HOSTNAME="$1"
if [ -z "$HOSTNAME" ]; then
	echo "usage: $0 hostname"
	exit 1
fi

cleanup() {
	rm -rf "$tmp"
}

makefile() {
	OWNER="$1"
	PERMS="$2"
	FILENAME="$3"
	cat > "$FILENAME"
	chown "$OWNER" "$FILENAME"
	chmod "$PERMS" "$FILENAME"
}

rc_add() {
	mkdir -p "$tmp"/etc/runlevels/"$2"
	ln -sf /etc/init.d/"$1" "$tmp"/etc/runlevels/"$2"/"$1"
}

tmp="$(mktemp -d)"
trap cleanup EXIT

mkdir -p "$tmp"/etc
makefile root:root 0644 "$tmp"/etc/hostname <<EOF
$HOSTNAME
EOF

makefile root:root 0644 "$tmp"/etc/issue <<EOF
Welcome to RiverMeadow Helper Appliance
EOF

cp /aports/scripts/traffic-filter.conf "$tmp"/etc/traffic-filter.conf

cp /aports/scripts/rmstool "$tmp"/etc/rmstool
chmod 755 "$tmp"/etc/rmstool

# create a service file for rmstool
mkdir -p "$tmp"/etc/init.d
makefile root:root 0755 "$tmp"/etc/init.d/rmstool <<'EOF'	
#!/sbin/openrc-run

command="/etc/rmstool"
pidfile="/var/run/rmstool.pid"
name="rmstool"
description="RMS Tool Service"
depend() {
	after net
}
EOF

# mkdir -p "$tmp"/etc/network
# makefile root:root 0644 "$tmp"/etc/network/interfaces <<EOF
# auto lo
# iface lo inet loopback

# auto eth0
# iface eth0 inet static
# EOF

mkdir -p "$tmp"/etc/apk
makefile root:root 0644 "$tmp"/etc/apk/world <<EOF
alpine-baselayout
busybox
nftables
doas
qemu-guest-agent
linux-virt
EOF

rc_add devfs sysinit
rc_add dmesg sysinit
rc_add hwdrivers sysinit
rc_add modloop sysinit

rc_add hwclock boot
rc_add modules boot
rc_add sysctl boot
rc_add hostname boot
rc_add bootmisc boot
rc_add syslog boot
rc_add qemu-guest-agent boot
rc_add klogd boot
rc_add nftables boot
rc_add doas boot
rc_add local default

rc_add mount-ro shutdown
rc_add killprocs shutdown
rc_add savecache shutdown

tar -c -C "$tmp" etc | gzip -9n > $HOSTNAME.apkovl.tar.gz