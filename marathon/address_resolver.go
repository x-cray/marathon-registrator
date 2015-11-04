package marathon

import (
	"net"

	log "github.com/Sirupsen/logrus"
)

type AddressResolver interface {
	Resolve(hostname string) (string, error)
}

type defaultAddressResolver struct{}

func (r defaultAddressResolver) Resolve(hostname string) (string, error) {
	address, err := net.ResolveIPAddr("ip", hostname)
	if err != nil {
		log.WithFields(log.Fields{
			"prefix":   "resolver",
			"hostname": hostname,
			"err":      err,
		}).Warn("Unable to resolve address")
		return "", err
	}

	return address.IP.String(), nil
}
