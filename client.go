package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/tun"
	"log"
	"net"
)

func startClient(config *core.Config) {
	// create cipher
	cp, err := core.NewCipher(config.Cipher, config.Passwd)
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
	conn, err := net.Dial("tcp", config.Address)
	if err != nil {
		log.Fatalf("cannot connect to server: %v\n", config.Address)
	}
	log.Printf("server connected: %v\n", conn.RemoteAddr().String())

	// cipher wrapped
	conn = core.NewStreamConn(conn, cp)

	buf := make(nat.IPPacket, 64*1024)

	for {
		// read TUN to a large buffer
		l, err := device.Read(buf)
		if err != nil {
			log.Printf("failed to read IPPacket from TUN device: %v\n", err)
			break
		}
		// skip TUN PI head
		ipp := buf[4:l]
		log.Printf("IPPacket:% x\n", ipp)
		// check IPPacket.Length()
		if pl, _ := ipp.Length(); pl != len(ipp) {
			log.Printf("IPPacket Lenght() mismatch: %v\n", pl)
			break
		}
		// log
		src, _ := ipp.GetIP(nat.SourceIP)
		dst, _ := ipp.GetIP(nat.DestinationIP)
		log.Printf("IPPacket read: Version: %v, Length: %v, Source: %v, Destination: %v", ipp.Version(), len(ipp), src.String(), dst.String())

		// write
		if _, err = conn.Write(ipp); err != nil {
			log.Printf("failed to send IPPacket to server: %v: %v\n", conn.RemoteAddr().String(), err)
			break
		}
	}
}
