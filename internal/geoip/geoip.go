package geoip

import (
	"context"
	_ "embed"
	"errors"
	"net"
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

// https://github.com/geoacumen/geoacumen-country
//
//go:generate sh -c "curl -L --output countries.mmdb https://github.com/geoacumen/geoacumen-country/raw/refs/heads/master/Geoacumen-Country.mmdb"
//go:embed countries.mmdb
var countries []byte
var geoDB *maxminddb.Reader

var (
	ErrInvalidIP = errors.New("invalid ip")
	ErrLookup    = errors.New("error trying to lookup address")
)

type Record struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
}

func Lookup(ctx context.Context, address string) (Record, error) {
	var record Record

	parsedIP, err := netip.ParseAddr(address)
	if err != nil {
		ips, errHost := net.DefaultResolver.LookupHost(ctx, address)
		if errHost != nil {
			return record, errors.Join(errHost, ErrInvalidIP)
		}

		parsedIP, err = netip.ParseAddr(ips[0])
		if err != nil {
			return record, errors.Join(err, ErrInvalidIP)
		}
	}

	if err = geoDB.Lookup(parsedIP).Decode(&record); err != nil {
		return record, errors.Join(err, ErrLookup)
	}

	return record, nil
}

func init() { //nolint:gochecknoinits
	reader, err := maxminddb.OpenBytes(countries)
	if err != nil {
		panic(err)
	}

	geoDB = reader
}
