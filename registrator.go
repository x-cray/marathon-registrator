package main

import (
	"errors"
	"log/syslog"
	"os"

	"github.com/x-cray/marathon-service-registrator/bridge"
	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	logrusSyslog "github.com/Sirupsen/logrus/hooks/syslog"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	version        = "0.0.1"
	app            = kingpin.New("registrator", "Automatically registers/deregisters Marathon tasks as services in Consul.").Version(version)
	consul         = app.Flag("consul", "Address and port of Consul agent").Short('c').Default("http://127.0.0.1:8500").URL()
	marathon       = app.Flag("marathon", "URL of Marathon instance. Multiple inctances may be specified in case of HA setup: http://addr1:8080,addr2:8080,addr3:8080").Short('m').Default("http://127.0.0.1:8080").String()
	resyncInterval = app.Flag("resync-interval", "Time interval to resync Marathon services to determine dangling instances. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"").Short('i').Default("5m").Duration()
	enableDryRun   = app.Flag("dry-run", "Do not perform actual service registeration/deregistration").Short('d').Bool()
	logLevel       = app.Flag("log-level", "Set the logging level - valid values are \"debug\", \"info\" (default), \"warn\", \"error\", and \"fatal\"").Short('l').Default("info").Enum("debug", "info", "warn", "error", "fatal")
	enableSyslog   = app.Flag("syslog", "Send the log output to syslog").Short('s').Bool()
)

func validateParams(app *kingpin.Application) error {
	if *resyncInterval <= 0 {
		return errors.New("--resync-interval must be greater than 0")
	}
	return nil
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	config, err := getConfig()
	assert(err)

	log.Infof("Starting Marathon service registrator: %v", version)
	b, err := bridge.New(config)
	assert(err)

	log.Info("Performing initial sync")
	syncErr := b.Sync()
	if syncErr != nil {
		log.Errorf("Failed to sync services: %v", syncErr)
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
		DryRun:         *enableDryRun,
	}

	// Setup the logging.
	if level, err := log.ParseLevel(*logLevel); err != nil {
		return nil, err
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
		log.SetLevel(level)
	}

	if *enableSyslog {
		if hook, err := logrusSyslog.NewSyslogHook("", "", syslog.LOG_DEBUG, app.Name); err != nil {
			return nil, err
		} else {
			log.AddHook(hook)
		}
	}

	return c, nil
}
