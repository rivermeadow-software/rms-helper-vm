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

mkdir -p "$tmp"/home/appliance
mkdir -p "$tmp"/usr/local/bin

makefile root:root 0644 "$tmp"/etc/passwd <<'EOF'
root:x:0:0:root:/root:/bin/sh
appliance:x:1000:1000:Appliance User:/home/appliance:/bin/sh
EOF

makefile root:root 0644 "$tmp"/etc/group <<'EOF'
root:x:0:
appliance:x:1000:
EOF

makefile root:root 0644 "$tmp"/etc/shadow <<'EOF'
root:*:19700:0:99999:7:::
appliance:*:19700:0:99999:7:::
EOF

chown -R 1000:1000 "$tmp"/home/appliance

makefile root:root 0755 "$tmp"/etc/kiosk-launch <<'EOF'
#!/bin/sh

export TERM=xterm-256color
clear

#cd /home/appliance || exit 1

#exec su -s /bin/sh appliance -c /etc/rmstool

exec /etc/rmstool

EOF

makefile root:root 0644 "$tmp"/etc/inittab <<'EOF'
::sysinit:/sbin/openrc sysinit
::sysinit:/sbin/openrc boot
::wait:/sbin/openrc default
::shutdown:/sbin/openrc shutdown

tty1::respawn:/etc/kiosk-launch </dev/tty1 >/dev/tty1 2>&1

::shutdown:/bin/umount -a -r
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
rc_add local default

rc_add mount-ro shutdown
rc_add killprocs shutdown
rc_add savecache shutdown

tar -c -C "$tmp" etc | gzip -9n > $HOSTNAME.apkovl.tar.gz