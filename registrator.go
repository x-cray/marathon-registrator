package main

import (
	"log"
	"os"

	"github.com/x-cray/marathon-service-registrator/config"

	"github.com/hashicorp/consul-template/logging"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	version        = "0.0.1"
	app            = kingpin.New("registrator", "Automatically registers/deregisters Marathon tasks as services in Consul.").Version(version)
	consul         = app.Flag("consul", "Address and port of Consul agent").Short('c').Default("127.0.0.1:8500").String()
	marathon       = app.Flag("marathon", "URL of Marathon instance. Multiple inctances may be specified in case of HA setup: http://addr1:8080,addr2:8080,addr3:8080").Short('m').Default("http://127.0.0.1:8080").String()
	resyncInterval = app.Flag("resync-interval", "Time interval to resync Marathon services to determine dangling instances. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"").Short('i').Default("5m").Duration()
	logLevel       = app.Flag("log-level", "Set the logging level - valid values are \"debug\", \"info\", \"warn\" (default), and \"err\"").Short('l').Default("warn").Enum("debug", "info", "warn", "err")
	syslog         = app.Flag("syslog", "Send the output to syslog instead of standard error and standard out. The syslog facility defaults to LOCAL0 and can be changed using attribute").Short('s').Bool()
	syslogFacility = app.Flag("syslog-facility", "Set the facility where syslog should log. If this attribute is supplied, the --syslog flag must also be supplied").Short('f').Default("LOCAL0").String()
)

func main() {
	log.Printf("Starting Marathon service registrator: %v\n", version)

	_, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}
}

func getConfig() (*config.Config, error) {
	kingpin.VersionFlag.Short('v')
	kingpin.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	c := &config.Config{
		Consul:         *consul,
		Marathon:       *marathon,
		ResyncInterval: *resyncInterval,
	}

	// Setup the logging
	if err := logging.Setup(&logging.Config{
		Name:           app.Name,
		Level:          *logLevel,
		Syslog:         *syslog,
		SyslogFacility: *syslogFacility,
		Writer:         os.Stderr,
	}); err != nil {
		return nil, err
	}

	return c, nil
}
