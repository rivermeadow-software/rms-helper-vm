package main

import (
	"log"
	"os"

	"github.com/vishvananda/netlink"
)

func main() {
	// set up network interface (bring eth0 up)
	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		log.Println("Error getting eth0:", err)
		return
	}
	netlink.LinkSetUp(eth0)
	applyDHCP()
	// This is a placeholder main function. The actual logic for the NetFlow collector would go here.
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	tui()
}
