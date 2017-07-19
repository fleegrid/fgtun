package main

import (
	"flag"
	"fmt"
	"github.com/fleegrid/core"
	"log"
	"os"
	"os/signal"
)

var clientMode = false
var serverMode = false
var helpFlag = false

func main() {
	flag.BoolVar(&helpFlag, "h", false, "")
	flag.BoolVar(&helpFlag, "help", false, "show help")
	flag.BoolVar(&clientMode, "c", false, "")
	flag.BoolVar(&clientMode, "client", false, "client mode")
	flag.BoolVar(&serverMode, "s", false, "")
	flag.BoolVar(&serverMode, "server", false, "server mode")
	flag.Parse()

	var url string

	if len(flag.Args()) > 0 {
		url = flag.Args()[0]
	} else {
		url = os.Getenv("FLEE_URL")
	}

	if len(url) == 0 {
		showHelp()
		return
	}

	config, err := core.ParseConfigFromURL(url)

	if err != nil {
		log.Fatalf("Failed to parse URL: %v\n", err)
	}

	if helpFlag {
		showHelp()
	} else if clientMode {
		log.Printf("fgtun v%v, FleeGrid as TUN device\n", Version)
		startClient(config)
	} else if serverMode {
		log.Printf("fgtun v%v, FleeGrid as TUN device\n", Version)
		startServer(config)
	} else {
		showHelp()
	}
}

func showHelp() {
	fmt.Printf("fgtun v%v\n", Version)
	fmt.Printf("Usage:\n")
	fmt.Printf("  Server Mode: fgtun -s [FLEE_URL], or use environment variable $FLEE_URL\n")
	fmt.Printf("  Client Mode: fgtun -c [FLEE_URL], or use environment variable $FLEE_URL\n")
	fmt.Printf("Option:\n")
	flag.PrintDefaults()
}

func startClient(config *core.Config) {
	var c *Client
	var err error

	// create signal chan
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// create client
	if c, err = NewClient(config); err != nil {
		os.Exit(1)
		return
	}

	// capture signal to stop
	go func() {
		<-signalChan
		c.Stop()
	}()

	// start
	c.Run()
}

func startServer(config *core.Config) {
	var s *Server
	var err error

	// create signal chan
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// create client
	if s, err = NewServer(config); err != nil {
		os.Exit(1)
		return
	}

	// capture signal to stop
	go func() {
		<-signalChan
		s.Stop()
	}()

	// start
	s.Run()
}
