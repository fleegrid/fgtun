package main

import (
	"flag"
	"fmt"
	"github.com/fleegrid/fgtun"
	"log"
)

var urlForServer = ""
var urlForClient = ""
var helpFlag = false

func main() {
	flag.StringVar(&urlForServer, "s", "", "start server")
	flag.StringVar(&urlForClient, "c", "", "start client")
	flag.BoolVar(&helpFlag, "h", false, "print help")
	flag.Parse()

	if helpFlag {
		printHelp()
	} else if len(urlForServer) > 0 {
		startServer(urlForServer)
	} else if len(urlForClient) > 0 {
		startClient(urlForClient)
	} else {
		printHelp()
	}
}

func startClient(url string) {
	log.Println("fgtun v" + fgtun.Version + ", starting as client")
}

func startServer(url string) {
	log.Println("fgtun v" + fgtun.Version + ", starting as server")
}

func printHelp() {
	fmt.Printf("fgtun v%v, FleeGrid as TUN device\n", fgtun.Version)
	fmt.Printf("Server:\n  fgtun -s [BIND_URL]\nClient:\n  fgtun -c [SERVER_URL]")
}
