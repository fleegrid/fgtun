// +build darwin

package main

import (
	"log"
	"net"
	"os/exec"
	"strings"
)

func (c *Client) setupTUN() (err error) {
	log.Println("tun: setting up TUN with 'ifconfig'")
	args := []string{
		c.device.Name(),
		c.localIP.String(),
		c.net.GatewayIP.String(),
		"netmask",
		net.IP(c.net.Mask).String(),
		"mtu",
		"1500",
		"up",
	}
	log.Printf("tun: executing 'ifconfig %v'\n", strings.Join(args, " "))
	if _, err = exec.Command("ifconfig", args...).Output(); err != nil {
		log.Println("tun: failed to execute 'ifconfig'")
		log.Printf("%s", err.(*exec.ExitError).Stderr)
	}
	return
}
