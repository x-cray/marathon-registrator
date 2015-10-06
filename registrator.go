package main

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app      = kingpin.New("registrator", "Automatically registers/deregisters Marathon tasks as services in Consul.")
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()

	fmt.Printf("Starting Marathon service registrator")
}
