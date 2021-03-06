package main

import (
	"errors"
	"fmt"
	"log/syslog"
	"os"
	"time"

	"github.com/x-cray/marathon-registrator/bridge"
	"github.com/x-cray/marathon-registrator/types"

	log "github.com/Sirupsen/logrus"
	logrusSyslog "github.com/Sirupsen/logrus/hooks/syslog"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	reconnectInterval = 5 * time.Second
)

var (
	version        string
	app            = kingpin.New("registrator", "Automatically registers/deregisters Marathon tasks as services in Consul.")
	consul         = app.Flag("consul", "Address and port of Consul agent").Short('c').Default("http://127.0.0.1:8500").URL()
	marathon       = app.Flag("marathon", "URL of Marathon instance. Multiple instances may be specified in case of HA setup: http://addr1:8080,addr2:8080,addr3:8080").Short('m').Default("http://127.0.0.1:8080").String()
	resyncInterval = app.Flag("resync-interval", "Time interval to resync Marathon services to determine dangling instances. Valid time units are \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\", \"h\"").Short('i').Default("5m").Duration()
	enableDryRun   = app.Flag("dry-run", "Do not perform actual service registration/deregistration. Just log intents").Short('d').Bool()
	logLevel       = app.Flag("log-level", "Set the logging level - valid values are \"debug\", \"info\", \"warn\", \"error\", and \"fatal\"").Short('l').Default("info").Enum("debug", "info", "warn", "error", "fatal")
	enableSyslog   = app.Flag("syslog", "Send the log output to syslog").Short('s').Bool()
	forceColors    = app.Flag("force-colors", "Force colored log output").Short('r').Bool()
)

func validateParams(app *kingpin.Application) error {
	if *resyncInterval <= 0 {
		return errors.New("--resync-interval must be greater than 0")
	}
	return nil
}

func printVersion(*kingpin.ParseContext) error {
	fmt.Fprintln(os.Stderr, version)
	os.Exit(0)
	return nil
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func trySync(b *bridge.Bridge) bool {
	if err := b.Sync(); err != nil {
		log.Errorf("Failed to sync services: %v", err)
		return false
	}

	return true
}

func main() {
	config, err := getConfig()
	assert(err)

	log.Infof("Starting Marathon service registrator v%s", version)
	b, err := bridge.New(config)
	assert(err)

	log.Info("Performing initial sync")
	for {
		if trySync(b) {
			break
		}
		log.Infof("Retrying initial sync in %v", reconnectInterval)
		time.Sleep(reconnectInterval)
	}

	quit := make(chan bool)

	// Start the resync timer.
	ticker := time.NewTicker(config.ResyncInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				trySync(b)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	// Run the main event application loop.
	for {
		err = b.ProcessSchedulerEvents()
		if err == nil {
			break
		}
		log.Info("Retrying event stream connection in %v", reconnectInterval)
		time.Sleep(reconnectInterval)
	}

	log.Warn("Scheduler event loop closed")
	close(quit)
}

func getConfig() (*types.Config, error) {
	app.Validate(validateParams)
	kingpin.HelpFlag.Short('h')
	app.Flag("version", "Print application version and exit").PreAction(printVersion).Short('v').Bool()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	c := &types.Config{
		Consul:         *consul,
		Marathon:       *marathon,
		ResyncInterval: *resyncInterval,
		DryRun:         *enableDryRun,
	}

	// Setup the logging.
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		return nil, err
	}

	log.SetFormatter(&prefixed.TextFormatter{
		ForceColors: *forceColors,
	})
	log.SetLevel(level)

	if *enableSyslog {
		hook, err := logrusSyslog.NewSyslogHook("", "", syslog.LOG_DEBUG, app.Name)
		if err != nil {
			return nil, err
		}

		log.AddHook(hook)
	}

	return c, nil
}
