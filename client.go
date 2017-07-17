package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/tun"
	"io"
	"log"
	//"net"
)

func startClient(config *core.Config) {
	// create cipher
	_, err := core.NewCipher(config.Cipher, config.Passwd)
	if err != nil {
		log.Fatalf("failed to initializae cipher %v: %v\n", config.Cipher, err)
	}
	log.Printf("using cipher: %v\n", config.Cipher)
	// create TUN
	device, err := tun.NewDevice()
	if err != nil {
		log.Fatalf("failed to create TUN device: %v\n", err)
	}
	log.Printf("TUN device created: %v\n", device.Name())
	// create virtual subnet
	mnet, err := nat.NewNetFromCIDR(subnet)
	if err != nil {
		log.Fatalf("failed to create managed net: %v\n", err)
	}
	log.Printf("managed network created: %v --> %v\n", mnet.String(), mnet.GatewayIP.String())
	mnet6, err := nat.NewNetFromCIDR(subnet6)
	if err != nil {
		log.Fatalf("failed to create managed net: %v\n", err)
	}
	log.Printf("managed network created: %v --> %v\n", mnet6.String(), mnet6.GatewayIP.String())
	// dial
	/*
		conn, err := net.Dial("tcp", config.Address)
		if err != nil {
			log.Fatalf("cannot connect to server: %v\n", config.Address)
		}
	*/

	// cipher wrapped
	//conn = core.NewStreamConn(conn, cp)

	pinfo := make([]byte, 4)

	for {
		// read and drop TUN_PI
		if _, err := io.ReadFull(device, pinfo); err != nil {
			log.Printf("failed to drop TUN_PI: %v\n", err)
			break
		}
		// read TUN
		ipp, err := nat.ReadIPPacket(device)
		if err != nil {
			log.Printf("failed to read IPPacket from TUN device: %v\n", err)
			break
		}
		log.Printf("IPPacket:% x\n", ipp)
		src, _ := ipp.GetIP(nat.SourceIP)
		dst, _ := ipp.GetIP(nat.DestinationIP)
		log.Printf("IPPacket read: Version: %v, Length: %v, Source: %v, Destination: %v", ipp.Version(), len(ipp), src.String(), dst.String())
		// write
		//conn.Write(ipp)
	}
}
