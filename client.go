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
	var cp core.Cipher
	if cp, err = core.NewCipher(config.Cipher, config.Passwd); err != nil {
		log.Println("failed to initialize cipher")
		return
	}
	log.Printf("using cipher: %v\n", config.Cipher)

	// create TUN
	var device *tun.Device
	if device, err = tun.NewDevice(); err != nil {
		log.Println("failed to create TUN device")
		return
	}
	log.Printf("TUN device created: %v\n", device.Name())

	// dial
	var conn net.Conn
	if conn, err = net.Dial("tcp", config.Address); err != nil {
		log.Println("failed to connect server")
		return
	}
	log.Printf("server connected: %v\n", conn.RemoteAddr().String())

	// wrap net.Conn with cipher
	conn = core.NewStreamConn(conn, cp)

	// write loop
	go func() {
		for {
			var err error
			// read a IPPacket from server
			var ipp pkt.IPPacket
			if ipp, err = pkt.ReadIPPacket(conn); err != nil {
				log.Printf("Failed to read a IPPacket from server: %v\n", conn.RemoteAddr().String())
				break
			}
			// build TUNPacket
			tp := make(pkt.TUNPacket, len(ipp)+4)
			if ipp.Version() == 4 {
				tp.SetProto(syscall.AF_INET)
			} else {
				tp.SetProto(syscall.AF_INET6)
			}
			tp.CopyPayload(ipp)
			// write a IPPacket once a time
			if _, err := device.Write(tp); err != nil {
				log.Printf("Failed to write a IPPacket to TUN device: %v\n", device.Name())
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
		if l, err = device.Read(buf); err != nil {
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
		ipp := pkt.IPPacket(b)

		// check IPPacket.Length()
		var pl int
		if pl, err = ipp.Length(); err != nil {
			log.Println("failed to get IPPacket length")
			break
		}
		if pl != len(ipp) {
			log.Println("IPPacket length mismatch")
			err = errors.New("IPPacket length mismatch")
			break
		}

		// log
		src, _ := ipp.IP(pkt.SourceIP)
		dst, _ := ipp.IP(pkt.DestinationIP)
		log.Printf("IPPacket read: Version: %v, Length: %v, Source: %v, Destination: %v", ipp.Version(), len(ipp), src.String(), dst.String())

		// write
		if _, err = conn.Write(ipp); err != nil {
			log.Printf("failed to send IPPacket to server: %v: %v\n", conn.RemoteAddr().String(), err)
			break
		}
	}
	return
}
