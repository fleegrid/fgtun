package main

import (
	"errors"
	"github.com/fleegrid/core"
	"github.com/fleegrid/pkt"
	"github.com/fleegrid/tun"
	"log"
	"net"
	"syscall"
)

func startClient(config *core.Config) (err error) {
	// create cipher
	var cf core.Cipher
	if cf, err = core.NewCipher(config.Cipher, config.Passwd); err != nil {
		log.Println("failed to initialize cipher")
		return
	}
	log.Printf("using cipher: %v\n", config.Cipher)

	// create TUN
	var d *tun.Device
	if d, err = tun.NewDevice(); err != nil {
		log.Println("failed to create TUN device")
		return
	}
	log.Printf("TUN device created: %v\n", d.Name())

	// dial
	var conn net.Conn
	if conn, err = net.Dial("tcp", config.Address); err != nil {
		log.Println("failed to connect server")
		return
	}
	log.Printf("server connected: %v\n", conn.RemoteAddr().String())

	// wrap net.Conn with cipher
	conn = core.NewStreamConn(conn, cf)

	// setup TUN device
	if err = setupClientTUN(d.Name()); err != nil {
		log.Println("failed to setup TUN device")
		return
	}

	// write loop
	go func() {
		for {
			var err error
			// read a IPPacket from server
			var p pkt.IPPacket
			if p, err = pkt.ReadIPPacket(conn); err != nil {
				log.Printf("Failed to read a IPPacket from server: %v\n", conn.RemoteAddr().String())
				break
			}
			// build TUNPacket
			tp := make(pkt.TUNPacket, len(p)+4)
			if p.Version() == 4 {
				tp.SetProto(syscall.AF_INET)
			} else {
				tp.SetProto(syscall.AF_INET6)
			}
			tp.CopyPayload(p)
			// write a IPPacket once a time
			if _, err := d.Write(tp); err != nil {
				log.Printf("Failed to write a IPPacket to TUN device: %v\n", d.Name())
				break
			}
		}
	}()

	// large buffer
	buf := make([]byte, 64*1024)

	// read loop
	for {
		// read to buffer
		var l int
		if l, err = d.Read(buf); err != nil {
			log.Println("failed to read bytes from TUN device")
			break
		}

		// wrap bytes with TUNPacket
		tp := pkt.TUNPacket(buf[:l])

		// extract payload
		var b []byte
		if b, err = tp.Payload(); err != nil {
			log.Println("failed to extract payload from TUN device")
			break
		}

		// create IPPacket
		p := pkt.IPPacket(b)

		// check IPPacket.Length()
		var pl int
		if pl, err = p.Length(); err != nil {
			log.Println("failed to get IPPacket length")
			break
		}
		if pl != len(p) {
			log.Println("IPPacket length mismatch")
			err = errors.New("IPPacket length mismatch")
			break
		}

		// log
		src, _ := p.IP(pkt.SourceIP)
		dst, _ := p.IP(pkt.DestinationIP)
		log.Printf("IPPacket read: Version: %v, Length: %v, Source: %v, Destination: %v", p.Version(), len(p), src.String(), dst.String())

		// write
		if _, err = conn.Write(p); err != nil {
			log.Printf("failed to send IPPacket to server: %v: %v\n", conn.RemoteAddr().String(), err)
			break
		}
	}
	return
}
