#!/bin/sh
profile_helper_kvm() {
    profile_standard
    # Customize kernel options, e.g., for specific hardware or boot parameters
    kernel_cmdline="console=tty0 console=ttyS0,115200 ipv6.disable=1"
    kernel_addons="" # Example: Add ZFS support
	kernel_flavors="virt"

    # Add desired packages
#    apks="alpine-base busybox qemu-guest-agent linux-lts bash nftables doas"
#    apks="$apks busybox doas alpine-baselayout qemu-guest-agent nftables linux-virt"
    apks="$apks busybox doas alpine-base qemu-guest-agent nftables linux-virt"
    # Specify your custom overlay script
    apkovl="aports/scripts/genapkovl-mkimgoverlay.sh"
}

profile_demo() {
    profile_standard
    # Customize kernel options, e.g., for specific hardware or boot parameters
    kernel_cmdline="console=tty0 console=ttyS0,115200 ipv6.disable=1"
    kernel_addons="" # Example: Add ZFS support
	kernel_flavors="virt"

    # Add desired packages
    apks="$apks alpine-base qemu-guest-agent udev iptables iproute2 tcpdump sysklogd-openrc linux-virt bash bind conntrack-tools openssh" # Example: Add vim and open-vm-tools

    # Specify your custom overlay script
    apkovl="aports/scripts/genapkovl-mkimgoverlay.sh"
}

profile_helper_vmware() {
    profile_base
    # Customize kernel options, e.g., for specific hardware or boot parameters
    kernel_cmdline="console=tty0 console=ttyS0,115200 ipv6.disable=1"
    kernel_addons="" # Example: Add ZFS support
	#kernel_flavors="virt"

    # Add desired packages
    apks="$apks alpine-base open-vm-tools linux-virt nftables doas"

    # Specify your custom overlay script
    apkovl="aports/scripts/genapkovl-mkimgoverlay.sh"
}