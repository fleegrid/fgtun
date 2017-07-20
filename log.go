package main

import (
	"log"
)

var debugLog = true

func logln(a ...interface{}) {
	log.Println(a...)
}

func logf(s string, a ...interface{}) {
	log.Printf(s, a...)
}

func dlogln(a ...interface{}) {
	if debugLog {
		log.Println(append([]interface{}{"[DEBUG]"}, a...)...)
	}
}

func dlogf(s string, a ...interface{}) {
	if debugLog {
		log.Printf("[DEBUG] "+s, a...)
	}
}
