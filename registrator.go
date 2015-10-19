package main

import (
	"errors"
	"log"
	"os"

	"github.com/x-cray/marathon-service-registrator/types"
	"github.com/x-cray/marathon-service-registrator/bridge"

	"github.com/hashicorp/consul-template/logging"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	version        = "0.0.1"
	app            = kingpin.New("registrator", "Automatically registers/deregisters Marathon tasks as services in Consul.").Version(version)
	consul         = app.Flag("consul", "Address and port of Consul agent").Short('c').Default("http://127.0.0.1:8500").URL()
	marathon       = app.Flag("marathon", "URL of Marathon instance. Multiple inctances may be specified in case of HA setup: http://addr1:8080,addr2:8080,addr3:8080").Short('m').Default("http://127.0.0.1:8080").String()
	resyncInterval = app.Flag("resync-interval", "Time interval to resync Marathon services to determine dangling instances. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"").Short('i').Default("5m").Duration()
	dryRun         = app.Flag("dry-run", "Do not perform actual service registeration/deregistration").Short('d').Bool()
	logLevel       = app.Flag("log-level", "Set the logging level - valid values are \"debug\", \"info\", \"warn\" (default), and \"err\"").Short('l').Default("warn").Enum("debug", "info", "warn", "err")
	syslog         = app.Flag("syslog", "Send the output to syslog instead of standard error and standard out. The syslog facility defaults to LOCAL0 and can be changed using attribute").Short('s').Bool()
	syslogFacility = app.Flag("syslog-facility", "Set the facility where syslog should log. If this attribute is supplied, the --syslog flag must also be supplied").Short('f').Default("LOCAL0").String()
)

func validateParams(app *kingpin.Application) error {
	if *resyncInterval <= 0 {
		return errors.New("--resync-interval must be greater than 0")
	}
	return nil
}

func assert(err error) {
	if err != nil {
		app.FatalIfError(err, "")
	}
}

func main() {
	log.Printf("Starting Marathon service registrator: %v\n", version)

	config, err := getConfig()
	assert(err)

	b, err := bridge.New(config)
	assert(err)

	log.Println("Performing initial sync")
	syncErr := b.Sync()
	if syncErr != nil {
		log.Println("Failed to sync services:", syncErr)
	}

	b.ListenForEvents()
}

func getConfig() (*types.Config, error) {
	app.Validate(validateParams)
	kingpin.VersionFlag.Short('v')
	kingpin.HelpFlag.Short('h')
	kingpin.MustParse(app.Parse(os.Args[1:]))

	c := &types.Config{
		Consul:         *consul,
		Marathon:       *marathon,
		ResyncInterval: *resyncInterval,
		DryRun:         *dryRun,
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
