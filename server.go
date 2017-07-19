package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/pkt"
	"github.com/fleegrid/tun"
	"net"
)

// DefaultServerSubnet default subnet for server
const DefaultServerSubnet = "10.152.219.2/24"

// Server Server context
type Server struct {
	// init
	config  *core.Config
	cipher  core.Cipher
	net     *nat.Net
	localIP net.IP

	// boot
	tun      *tun.Device
	listener net.Listener

	// running

	// done
	done chan bool
}

// NewServer create a new server instance
func NewServer(config *core.Config) (s *Server, err error) {
	// alloc
	s = &Server{
		config: config,
		done:   make(chan bool, 2),
	}

	// create cipher
	if s.cipher, err = core.NewCipher(config.Cipher, config.Passwd); err != nil {
		logln("core: failed to initializae cipher:", config.Cipher, err)
		return
	}

	// create managed net
	if s.net, err = nat.NewNetFromCIDR(DefaultServerSubnet); err != nil {
		logln("nat: failed to create managed subnet:", err)
		return
	}

	// take a localIP
	if s.localIP, err = s.net.Take(); err != nil {
		logln("nat: failed to take a localIP:", err)
		return
	}

	return
}

// Run run the server, method not returns until Stop() called
func (s *Server) Run() (err error) {
	if err = s.boot(); err != nil {
		return
	}

	go s.acceptLoop()
	go s.tunReadLoop()

	<-s.done
	<-s.done
	return
}

func (s *Server) boot() (err error) {
	// TUN
	if s.tun, err = tun.NewDevice(); err != nil {
		logln("tun: failed to create device:", err)
		return
	}
	logln("tun: device created:", s.tun.Name())

	// listen
	if s.listener, err = net.Listen("tcp", s.config.Address); err != nil {
		logln("conn: failed to listen:", s.config.Address, err)
		return
	}
	logln("conn: listening on:", s.config.Address)

	// setup TUN
	if err = s.setupTUN(); err != nil {
		logln("tun: failed to setup TUN:", err)
		return
	}
	logln("tun: setup success")
	return
}

func (s *Server) acceptLoop() {
	// defer to notify done
	defer func() {
		s.done <- true
	}()

	// accept
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			logln("conn: failed to accept:", err)
			break
		}

		logf("conn: new connection:", conn.RemoteAddr().String())

		// cipher wrapped connection
		conn = core.NewStreamConn(conn, s.cipher)

		// handle connection
		go s.handleConnection(conn)
	}
}

func (s *Server) tunReadLoop() {
	// defer to notify done
	defer func() {
		s.done <- true
	}()
}

func (s *Server) setupTUN() (err error) {
	return
}

func (s *Server) shutdownTUN() (err error) {
	return
}

// handleConnection handles a connection between FleeGrid client and server
// it reads IPPacket sent from client, assigns and rewrite virtual IP and send to TUN device
func (s *Server) handleConnection(conn net.Conn) {
	// defer to close connection
	defer conn.Close()

	// extract RemoteAddr for name
	name := conn.RemoteAddr().String()

	// virtual ip
	var vip net.IP
	// original ip
	var oip net.IP
	// subnet
	var subnet *nat.Net

	// defer to remove virtual IP
	defer func() {
		if vip != nil && subnet != nil {
			subnet.Remove(vip)
		}
	}()

	for {
		ipp, err := pkt.ReadIPPacket(conn)
		if err != nil {
			logf("failed to read a IPPacket: %v: %v\n", name, err)
			break
		}
		// virtual ip not assigned
		if vip == nil {
			// determine subnet
			if ipp.Version() == 4 {
				subnet = s.net
			} else {
				logf("IPv6 is not supported")
				continue
			}
			// record orignal IP
			oip, err = ipp.IP(pkt.SourceIP)
			if err != nil {
				logf("cannot retrieve original IP: %v: %v\n", name, err)
				break
			}
			// take a virtual IP
			vip, err = subnet.Take()
			if err != nil {
				logf("cannot assign IP: %v: %v\n", name, err)
				break
			}
			logf("virtual IP assigned: %v: %v --> %v\n", name, oip.String(), vip.String())
			// rewrite IP
			err = ipp.SetIP(pkt.SourceIP, vip)
			if err != nil {
				logf("cannot rewrite source IP: %v: %v\n", name, err)
				break
			}
		} else {
			// check source IP
			noip, err := ipp.IP(pkt.SourceIP)
			if err != nil {
				logf("cannot retrieve original IP: %v: %v\n", name, err)
				break
			}
			if !noip.Equal(oip) {
				logf("original IP changed: %v: %v --> %v\n", name, oip.String(), noip.String())
				break
			}
			// rewrite IP
			err = ipp.SetIP(pkt.SourceIP, vip)
			if err != nil {
				logf("cannot rewrite source IP: %v: %v\n", name, err)
				break
			}
		}
		src, _ := ipp.IP(pkt.SourceIP)
		dst, _ := ipp.IP(pkt.DestinationIP)
		logf("IPPacket read: Version:%v, Length:%v, Source:%v, Destination:%v", ipp.Version(), len(ipp), src.String(), dst.String())
	}
}

// Stop stop the server, makes Run() exit
func (s *Server) Stop() {
}
