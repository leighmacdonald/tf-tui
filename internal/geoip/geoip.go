package geoip

import (
	_ "embed"
	"errors"
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

// https://github.com/geoacumen/geoacumen-country
//
//go:generate sh -c "curl -L --output countries.mmdb https://github.com/geoacumen/geoacumen-country/raw/refs/heads/master/Geoacumen-Country.mmdb"
//go:embed countries.mmdb
var countries []byte
var db *maxminddb.Reader

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

func Lookup(ipAddr string) (Record, error) {
	var record Record

	ip, err := netip.ParseAddr(ipAddr)
	if err != nil {
		return record, errors.Join(err, ErrInvalidIP)
	}

	if err = db.Lookup(ip).Decode(&record); err != nil {
		return record, errors.Join(err, ErrLookup)
	}

	return record, nil
}

func init() {
	reader, err := maxminddb.OpenBytes(countries)
	if err != nil {
		panic(err)
	}
	db = reader
}
