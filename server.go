package main

import (
	"github.com/fleegrid/core"
	"github.com/fleegrid/nat"
	"github.com/fleegrid/pkt"
	"github.com/fleegrid/tun"
	"log"
	"net"
)

const subnet = "192.168.220.1/24"
const subnet6 = "fe90::1/16"

// Server Server context
type Server struct {
	tun  *tun.Device
	net  *nat.Net
	net6 *nat.Net
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
			log.Printf("failed to read a IPPacket: %v: %v\n", name, err)
			break
		}
		// virtual ip not assigned
		if vip == nil {
			// determine subnet
			if ipp.Version() == 4 {
				subnet = s.net
			} else {
				subnet = s.net6
			}
			// record orignal IP
			oip, err = ipp.IP(pkt.SourceIP)
			if err != nil {
				log.Printf("cannot retrieve original IP: %v: %v\n", name, err)
				break
			}
			// take a virtual IP
			vip, err = subnet.Take()
			if err != nil {
				log.Printf("cannot assign IP: %v: %v\n", name, err)
				break
			}
			log.Printf("virtual IP assigned: %v: %v --> %v\n", name, oip.String(), vip.String())
			// rewrite IP
			err = ipp.SetIP(pkt.SourceIP, vip)
			if err != nil {
				log.Printf("cannot rewrite source IP: %v: %v\n", name, err)
				break
			}
		} else {
			// check source IP
			noip, err := ipp.IP(pkt.SourceIP)
			if err != nil {
				log.Printf("cannot retrieve original IP: %v: %v\n", name, err)
				break
			}
			if !noip.Equal(oip) {
				log.Printf("original IP changed: %v: %v --> %v\n", name, oip.String(), noip.String())
				break
			}
			// rewrite IP
			err = ipp.SetIP(pkt.SourceIP, vip)
			if err != nil {
				log.Printf("cannot rewrite source IP: %v: %v\n", name, err)
				break
			}
		}
		src, _ := ipp.IP(pkt.SourceIP)
		dst, _ := ipp.IP(pkt.DestinationIP)
		log.Printf("IPPacket read: Version:%v, Length:%v, Source:%v, Destination:%v", ipp.Version(), len(ipp), src.String(), dst.String())
	}
}

func startServer(config *core.Config) {
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

	// create server context
	server := &Server{
		tun:  device,
		net:  mnet,
		net6: mnet6,
	}

	// listen
	l, err := net.Listen("tcp", config.Address)
	if err != nil {
		log.Fatalf("failed to listen on %v: %v\n", config.Address, err)
	}
	log.Printf("listening on: %v\n", config.Address)
	// accept
	for {
		conn, err := l.Accept()

		if err != nil {
			log.Fatalf("failed to accept: %v\n", err)
			break
		}

		log.Printf("new connection: %v\n", conn.RemoteAddr().String())

		// cipher wrapped connection
		conn = core.NewStreamConn(conn, cp)

		// handle connection
		go server.handleConnection(conn)
	}
}
