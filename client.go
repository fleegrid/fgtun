package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/pkt"
	"github.com/fleegrid/tun"
	"log"
	"net"
	"syscall"
)

// DefaultClientSubnet default subnet for client, only two addresses will be taken
const DefaultClientSubnet = "10.152.219.1/24"

// Client represents a fgtun client
type Client struct {
	stop chan bool

	config  *core.Config
	cipher  core.Cipher
	net     *nat.Net
	localIP net.IP

	device *tun.Device
	conn   net.Conn
}

// NewClient creates a new client with config
func NewClient(config *core.Config) (c *Client, err error) {
	// create client
	c = &Client{
		config: config,
		stop:   make(chan bool, 1),
	}

	// create cipher
	if c.cipher, err = core.NewCipher(c.config.Cipher, c.config.Passwd); err != nil {
		log.Println("core: failed to initialize cipher")
		return
	}

	// create managed net
	if c.net, err = nat.NewNetFromCIDR(DefaultClientSubnet); err != nil {
		log.Println("nat: failed to create managed subnet")
		return
	}

	// assign a localIP
	if c.localIP, err = c.net.Take(); err != nil {
		log.Println("nat: faield to assign a localIP")
		return
	}

	return
}

// RemoteAddr get the string representation of remote addr
func (c *Client) RemoteAddr() string {
	if c.conn != nil {
		return c.conn.RemoteAddr().String()
	}
	return ""
}

// Run run the client
func (c *Client) Run() (err error) {
	// print informations
	log.Printf("core: using cipher %v\n", c.config.Cipher)
	log.Printf("nat: using subnet %v, %v -> %v\n", c.net.String(), c.localIP.String(), c.net.GatewayIP.String())

	// create TUN
	if c.device, err = tun.NewDevice(); err != nil {
		log.Println("tun: failed to create TUN device")
		return
	}
	log.Printf("tun: device created: %v\n", c.device.Name())

	// dial
	if c.conn, err = net.Dial("tcp", c.config.Address); err != nil {
		log.Println("conn: failed to connect server")
		return
	}
	log.Printf("conn: server connected: %v\n", c.RemoteAddr())

	// wrap net.Conn with cipher
	c.conn = core.NewStreamConn(c.conn, c.cipher)

	// setup TUN device
	if err = c.setupTUN(); err != nil {
		log.Println("failed to setup TUN device")
		return
	}
	log.Println("tun: setup complete")

	// write loop
	go func() {
		for {
			var err error
			// read a IPPacket from server
			var p pkt.IPPacket
			if p, err = pkt.ReadIPPacket(c.conn); err != nil {
				log.Printf("Failed to read a IPPacket from server: %v\n", c.conn.RemoteAddr().String())
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
			if _, err := c.device.Write(tp); err != nil {
				log.Printf("Failed to write a IPPacket to TUN device: %v\n", c.device.Name())
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
		if l, err = c.device.Read(buf); err != nil {
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
			log.Printf("IPPacket length mismatch: %v != %v\n", pl, len(p))
			continue
		}

		// log
		src, _ := p.IP(pkt.SourceIP)
		dst, _ := p.IP(pkt.DestinationIP)
		log.Printf("IPPacket read: Version: %v, Length: %v, Source: %v, Destination: %v", p.Version(), len(p), src.String(), dst.String())

		// write
		if _, err = c.conn.Write(p); err != nil {
			log.Printf("failed to send IPPacket to server: %v: %v\n", c.RemoteAddr(), err)
			break
		}
	}
	return
}
