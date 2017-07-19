// +build darwin

package main

import (
	"log"
	"net"
	"os/exec"
	"strings"
)

func (c *Client) setupTUN() (err error) {
	log.Println("tun: setting up TUN")
	// ifconfig
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
	// route
	args = []string{
		"add",
		"default",
		c.net.GatewayIP.String(),
		"-ifscope",
		c.device.Name(),
	}
	log.Printf("tun: executing 'route %v'\n", strings.Join(args, " "))
	if _, err = exec.Command("route", args...).Output(); err != nil {
		log.Println("tun: failed to execute 'route'")
		log.Printf("%s", err.(*exec.ExitError).Stderr)
	}
	return
}

func (c *Client) shutdownTUN() (err error) {
	log.Println("tun: shutting down TUN")
	// route
	args := []string{
		"delete",
		"default",
		c.net.GatewayIP.String(),
		"-ifscope",
		c.device.Name(),
	}
	log.Printf("tun: executing 'route %v'\n", strings.Join(args, " "))
	if _, err = exec.Command("route", args...).Output(); err != nil {
		log.Println("tun: failed to execute 'route'")
		log.Printf("%s", err.(*exec.ExitError).Stderr)
	}
	return
}
