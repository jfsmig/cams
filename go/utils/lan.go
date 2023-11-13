package utils

import (
	"github.com/juju/errors"
	"net"
)

func DiscoverSystemNics() ([]string, error) {
	var out []string
	itfs, err := net.Interfaces()
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, itf := range itfs {
		out = append(out, itf.Name)
	}
	return out, nil
}
