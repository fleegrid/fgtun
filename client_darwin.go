// +build darwin

package main

import (
	"bufio"
	"bytes"
	"net"
	"os/exec"
	"strings"
)

func (c *Client) setupTUN() (err error) {
	logln("tun: setting up TUN")
	// ifconfig, setup TUN device
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
	logf("tun: executing 'ifconfig %v'\n", strings.Join(args, " "))
	if _, err = exec.Command("ifconfig", args...).Output(); err != nil {
		logln("tun: failed to execute 'ifconfig'")
		logf("%s", err.(*exec.ExitError).Stderr)
		return
	}
	// route, get default route
	if c.lastGatewayIP, err = routeGetDefaultGateway(); err != nil {
		logln("tun: failed to get default gateway")
		return
	}
	if len(c.lastGatewayIP) != 0 {
		logf("tun: current GatewayIP recorded %s\n", c.lastGatewayIP)
		if err = routeDeleteDefaultGateway(); err != nil {
			logln("tun: failed to delete default gateway")
			return
		}
	} else {
		logln("tun: WARN! failed to get current GatewayIP")
	}
	// route, add default route
	err = routeAddDefaultGateway(c.net.GatewayIP.String())
	return
}

func (c *Client) shutdownTUN() (err error) {
	logln("tun: shutting down TUN")
	if len(c.lastGatewayIP) > 0 {
		// try delete default gateway ignoring error
		routeDeleteDefaultGateway()
		// add lastGatewayIP as gateway
		logf("tun: recovery gateway %v\n", c.lastGatewayIP)
		if err = routeAddDefaultGateway(c.lastGatewayIP); err != nil {
			logln("tun: failed to recovery default gateway")
			return
		}
	}
	return
}

func routeAddDefaultGateway(ip string) (err error) {
	args := []string{
		"add",
		"default",
		ip,
	}
	logf("tun: executing 'route %v'\n", strings.Join(args, " "))
	if _, err = exec.Command("route", args...).Output(); err != nil {
		logln("tun: failed to execute 'route'")
		logf("%s", err.(*exec.ExitError).Stderr)
	}
	return
}

func routeDeleteDefaultGateway() (err error) {
	args := []string{
		"delete",
		"default",
	}
	logf("tun: executing 'route %v'\n", strings.Join(args, " "))
	if _, err = exec.Command("route", args...).Output(); err != nil {
		logln("tun: failed to execute 'route'")
		logf("%s", err.(*exec.ExitError).Stderr)
	}
	return
}

func routeGetDefaultGateway() (ret string, err error) {
	args := []string{
		"-n",
		"get",
		"default",
	}
	var out []byte
	logf("tun: executing 'route %v'\n", strings.Join(args, " "))
	if out, err = exec.Command("route", args...).Output(); err != nil {
		logln("tun: failed to execute 'route'")
		logf("%s", err.(*exec.ExitError).Stderr)
		return
	}
	bout := bufio.NewReader(bytes.NewBuffer(out))
	for {
		var l []byte
		if l, _, err = bout.ReadLine(); err != nil {
			break
		}
		cs := strings.Split(strings.TrimSpace(string(l)), ":")
		if len(cs) == 2 && strings.EqualFold(strings.TrimSpace(cs[0]), "gateway") {
			ret = strings.TrimSpace(cs[1])
			break
		}
	}
	return
}
