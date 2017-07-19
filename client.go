package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/pkt"
	"github.com/fleegrid/tun"
	"net"
	"syscall"
)

// DefaultClientSubnet default subnet for client, only two addresses will be taken
const DefaultClientSubnet = "10.152.219.1/24"

// Client represents a fgtun client
type Client struct {
	config  *core.Config
	cipher  core.Cipher
	net     *nat.Net
	localIP net.IP

	device *tun.Device
	conn   net.Conn

	stopping bool
	done     chan bool
}

// NewClient creates a new client with config
func NewClient(config *core.Config) (c *Client, err error) {
	// create client
	c = &Client{
		config: config,
		done:   make(chan bool, 2),
	}

	// create cipher
	if c.cipher, err = core.NewCipher(c.config.Cipher, c.config.Passwd); err != nil {
		logln("core: failed to initialize cipher:", err)
		return
	}

	// create managed net
	if c.net, err = nat.NewNetFromCIDR(DefaultClientSubnet); err != nil {
		logln("nat: failed to create managed subnet:", err)
		return
	}

	// assign a localIP
	if c.localIP, err = c.net.Take(); err != nil {
		logln("nat: faield to assign a localIP:", err)
		return
	}

	return
}

// Run start the client
func (c *Client) Run() (err error) {
	// boot
	if err = c.boot(); err != nil {
		return
	}

	// read loop
	go c.readLoop()
	// write loop
	go c.writeLoop()

	// wait both loop done
	<-c.done
	<-c.done

	return
}

func (c *Client) boot() (err error) {
	// clear stopping
	c.stopping = false

	// print informations
	logf("core: using cipher %v\n", c.config.Cipher)
	logf("nat: using subnet %v, %v -> %v\n", c.net.String(), c.localIP.String(), c.net.GatewayIP.String())

	// create TUN
	if c.device, err = tun.NewDevice(); err != nil {
		logln("tun: failed to create TUN device:", err)
		return
	}
	logf("tun: device created: %v\n", c.device.Name())

	// dial
	if c.conn, err = net.Dial("tcp", c.config.Address); err != nil {
		logln("conn: failed to connect server:", err)
		return
	}
	logf("conn: server connected: %v\n", c.conn.RemoteAddr().String())

	// wrap net.Conn with cipher
	c.conn = core.NewStreamConn(c.conn, c.cipher)

	// setup TUN device
	if err = c.setupTUN(); err != nil {
		logln("tun: failed to setup TUN device:", err)
		return
	}
	logln("tun: setup complete")

	return
}

func (c *Client) readLoop() {
	// notify done
	defer func() {
		c.done <- true
	}()

	var err error

	// large buffer
	buf := make([]byte, 64*1024)

	// read loop
	for {
		// read to buffer
		var l int
		if l, err = c.device.Read(buf); err != nil {
			if !c.stopping {
				logln("tun: failed to read bytes from TUN device:", err)
			}
			break
		}

		// wrap bytes with TUNPacket
		tp := pkt.TUNPacket(buf[:l])

		// extract payload
		var b []byte
		if b, err = tp.Payload(); err != nil {
			logln("tun: failed to extract payload from TUN device", err)
			break
		}

		// create IPPacket
		p := pkt.IPPacket(b)

		// check IPPacket.Length()
		var pl int
		if pl, err = p.Length(); err != nil {
			logln("tun: failed to get IPPacket length:", err)
			break
		}
		if pl != len(p) {
			logf("tun: IPPacket length mismatch: %v != %v\n", pl, len(p))
			continue
		}

		// log
		src, _ := p.IP(pkt.SourceIP)
		dst, _ := p.IP(pkt.DestinationIP)
		dlogf("tun: IPPacket read: v%v, len: %v, src: %v, dst: %v", p.Version(), len(p), src.String(), dst.String())

		// write
		if _, err = c.conn.Write(p); err != nil {
			if !c.stopping {
				logln("conn: failed to send IPPacket to server:", err)
			}
			break
		}
	}
}

func (c *Client) writeLoop() {
	// notify done
	defer func() {
		c.done <- true
	}()

	var err error

	for {
		// read a IPPacket from server
		var p pkt.IPPacket
		if p, err = pkt.ReadIPPacket(c.conn); err != nil {
			if !c.stopping {
				logln("conn: failed to read a IPPacket:", err)
			}
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
		if _, err = c.device.Write(tp); err != nil {
			if !c.stopping {
				logln("conn: failed to write a IPPacket to TUN device:", err)
			}
			break
		}
	}
}

// Stop shutdown the client, makes Run() returns with nil error
func (c *Client) Stop() {
	// mark stopping
	c.stopping = true

	// close conn
	if c.conn != nil {
		logln("conn: shutting down")
		c.conn.Close()
	}

	// close tun
	if c.device != nil {
		logln("tun: shutting down")
		c.device.Close()
	}
}
