package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/pkt"
	"github.com/fleegrid/sh"
	"github.com/fleegrid/tun"
	"net"
	"syscall"
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
	// [NAT IP] -> [Client Virtual LocalIP]
	clientIPs *nat.IPMap

	// [NAT IP] -> [net.Conn]
	clients *ConnMgr

	// done
	done chan bool
	stop bool
}

// NewServer create a new server instance
func NewServer(config *core.Config) (s *Server, err error) {
	// alloc
	s = &Server{
		config:    config,
		clientIPs: nat.NewIPMap(),
		clients:   NewConnMgr(),
		done:      make(chan bool, 2),
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
	s.stop = false

	// print informations
	logf("core: using cipher %v\n", s.config.Cipher)
	logf("nat: using subnet %v, %v -> %v\n", s.net.String(), s.localIP.String(), s.net.GatewayIP.String())

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
		dlogln("server: acceptLoop done")
		s.done <- true
	}()

	// accept
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			if !s.stop {
				logln("conn: failed to accept:", err)
			}
			break
		}

		logln("conn: new connection:", conn.RemoteAddr().String())

		// cipher wrapped connection
		conn = core.NewStreamConn(conn, s.cipher)

		// handle connection
		go s.handleConnection(conn)
	}
}

func (s *Server) tunReadLoop() {
	// defer to notify done
	defer func() {
		dlogln("server: tunReadLoop done")
		s.done <- true
	}()

	buf := make([]byte, 64*1024)

	for {
		// read buf
		var l int
		var err error
		if l, err = s.tun.Read(buf); err != nil {
			if !s.stop {
				logln("tun: failed to read packet:", err)
			}
			break
		}

		// extract Payload
		var pl []byte
		if pl, err = pkt.TUNPacket(buf[:l]).Payload(); err != nil {
			logln("tun: failed to extract payload:", err)
			continue
		}

		// extract IPPacket
		ipp := pkt.IPPacket(pl)

		// check IP version
		if ipp.Version() != 4 {
			logln("tun: only IPv4 is supported")
			continue
		}

		// check IPPacket length
		if il, err := ipp.Length(); il != len(ipp) || err != nil {
			logln("tun: IPPacket length mismatch", il, "!=", len(ipp), err)
			continue
		}

		// extract destination IP
		dstIP, err := ipp.IP(pkt.DestinationIP)
		if err != nil {
			logln("tun: cannot extract destination IP")
			continue
		}

		// log
		srcIP, _ := ipp.IP(pkt.SourceIP)
		dlogf("tun: IPPacket read: v%v, len: %v, %v -> %v", ipp.Version(), len(ipp), srcIP.String(), dstIP.String())

		// get client localIP
		clientLocalIP := s.clientIPs.Get(dstIP)

		if clientLocalIP == nil {
			logln("tun: IPPacket destination IP not found")
			continue
		}

		// rewrite DestinationIP to client's virtual localIP
		if err := ipp.SetIP(pkt.DestinationIP, clientLocalIP); err != nil {
			logln("tun: IPPacket destination IP failed to set")
			continue
		}

		// find client connection
		conn := s.clients.Get(dstIP)

		if conn == nil {
			logln("conn: cannot find corresponding connection for", dstIP.String())
			continue
		}

		// make a local copy and write
		p := make([]byte, len(ipp), len(ipp))
		copy(p, ipp)
		go conn.Write(p)
	}
}

func (s *Server) setupTUN() (err error) {
	p := &sh.Params{}
	if _, err = sh.Run(serverSetupScript, p); err != nil {
		logln("tun: failed to setup device:", err)
	}
	return
}

func (s *Server) shutdownTUN() (err error) {
	p := &sh.Params{}
	if _, err = sh.Run(serverShutdownScript, p); err != nil {
		logln("tun: failed to shutdown device:", err)
	}
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

	// defer to remove virtual IP
	defer func() {
		if vip != nil {
			// remote conn ref
			s.clients.Delete(vip)
			// delete ip connection
			s.clientIPs.Delete(vip)
			// release allocated virtual IP
			s.net.Remove(vip)
		}
	}()

	for {
		// read packet
		ipp, err := pkt.ReadIPPacket(conn)
		if err != nil {
			if !s.stop {
				logln("conn:", name, "failed to read a IPPacket:", err)
			}
			break
		}

		// check version
		if ipp.Version() != 4 {
			logln("conn:", name, "IPPacket v6 is not supported")
			break
		}

		// virtual ip not assigned
		if vip == nil {

			// take a virtual IP
			vip, err = s.net.Take()
			if err != nil {
				logln("conn:", name, "cannot assign IP:", err)
				break
			}

			// get orignal IP
			oip, err = ipp.IP(pkt.SourceIP)
			if err != nil {
				logln("conn:", name, "cannot retrieve original IP:", name, err)
				break
			}

			// record [virtual IP] -> [client localIP]
			s.clientIPs.Set(vip, oip)
			// record [virtual IP] -> [net.Conn]
			s.clients.Set(vip, conn)

			logln("conn:", name, "virtual IP assigned", oip.String(), "->", vip.String())

			// rewrite IP
			err = ipp.SetIP(pkt.SourceIP, vip)
			if err != nil {
				logln("conn:", name, "cannot rewrite source IP:", err)
				break
			}
		} else {
			// check source IP
			noip, err := ipp.IP(pkt.SourceIP)
			if err != nil {
				logln("conn:", name, "cannot retrieve original IP:", err)
				break
			}
			if !noip.Equal(oip) {
				logln("conn:", name, "original IP changed:", oip.String(), "->", noip.String())
				break
			}

			// rewrite IP
			err = ipp.SetIP(pkt.SourceIP, vip)
			if err != nil {
				logln("conn:", name, "cannot rewrite source IP:", err)
				break
			}
		}

		// log
		src, _ := ipp.IP(pkt.SourceIP)
		dst, _ := ipp.IP(pkt.DestinationIP)
		dlogf("conn: %s IPPacket read: v%v, len:%v, %v -> %v\n", name, ipp.Version(), len(ipp), src.String(), dst.String())

		// create TUNPacket
		tp := make(pkt.TUNPacket, len(ipp)+4, len(ipp)+4)
		tp.SetProto(syscall.AF_INET)
		tp.CopyPayload(ipp)

		// write TUNPacket to tun
		if s.tun != nil {
			if _, err = s.tun.Write(tp); err != nil && !s.stop {
				logln("tun: failed to write:", err)
			}
		}
	}
}

// Stop stop the server, makes Run() exit
func (s *Server) Stop() {
	s.stop = true

	logln("tun: closing device")
	s.shutdownTUN()
	if s.tun != nil {
		s.tun.Close()
	}

	logln("conn: closing server")
	if s.listener != nil {
		s.listener.Close()
	}

	<-s.done
	<-s.done
}
