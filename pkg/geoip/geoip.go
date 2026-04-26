package geoip

import (
	_ "embed"
	"errors"
	"net"
	"strings"
	"sync"

	maxminddb "github.com/oschwald/maxminddb-golang"
)

//go:embed geoip.db
var db []byte

var (
	dbOnce = sync.OnceValues(func() (*maxminddb.Reader, error) {
		db, err := maxminddb.FromBytes(db)
		if err != nil {
			return nil, err
		}
		return db, nil
	})
)

func Lookup(ip net.IP) (string, error) {
	db, err := dbOnce()
	if err != nil {
		return "", err
	}

	var record map[string]any
	err = db.Lookup(ip, &record)
	if err != nil {
		return "", err
	}

	if code := countryCode(record); code != "" {
		return strings.ToLower(code), nil
	}

	return "", errors.New("IP not found")
}

func countryCode(record map[string]any) string {
	paths := [][]string{
		{"country"},
		{"country_code"},
		{"country", "iso_code"},
		{"registered_country", "iso_code"},
		{"continent"},
		{"continent_code"},
		{"continent", "code"},
	}

	for _, path := range paths {
		if code := strings.TrimSpace(stringAt(record, path...)); code != "" {
			return code
		}
	}
	return ""
}

func stringAt(record map[string]any, path ...string) string {
	var current any = record
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}
	return value
}
