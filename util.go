package main

import (
	"log"
	"os/exec"
	"strings"
)

const ifconfig = "ifconfig"

const clientLocalIP = "10.220.52.1"
const clientRemoteIP = "10.220.52.2"
const clientNetmask = "255.255.255.0"

func setupClientTUN(name string) (err error) {
	args := []string{
		name,
		clientLocalIP,
		clientRemoteIP,
		"netmask",
		clientNetmask,
		"up",
	}
	log.Printf("TUN: '%s %s'", ifconfig, strings.Join(args, " "))

	cmd := exec.Command(ifconfig, args...)
	var out []byte
	if out, err = cmd.Output(); err != nil {
		log.Printf("ifconfig: %s", out)
		return
	}
	log.Printf("TUN: device up %s --> %s\n", clientLocalIP, clientRemoteIP)
	return
}
