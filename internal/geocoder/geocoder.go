package geocoder

import (
	"context"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/oschwald/maxminddb-golang"
	"net"
)

// OfflineGeocoder represents an offline geocoder.
type OfflineGeocoder struct {
	MaxMindDB *maxminddb.Reader
	DB        *db.DB
}

// Result represents the geolocation data.
type Result struct {
	CountryCode    string
	CountryID      int64
	CountryCode3   string
	CityName       string
	RegionName     string
	RegionCode     string
	Lat            float64
	Lon            float64
	Accuracy       int
	ZipCode        string
	IPService      int
	UnknownCountry bool
}

type GeoData struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	City struct {
		Name string `maxminddb:"name"`
	} `maxminddb:"city"`
	Subdivisions struct {
		MostSpecific struct {
			Name    string `maxminddb:"name"`
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"most_specific"`
	} `maxminddb:"subdivisions"`
	Location struct {
		Latitude       float64 `maxminddb:"latitude"`
		Longitude      float64 `maxminddb:"longitude"`
		AccuracyRadius int     `maxminddb:"accuracy_radius"`
	} `maxminddb:"location"`
	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`
	Continent struct {
		Name string `maxminddb:"name"`
	}
}

const (
	MAX_MIND_PROVIDER_CODE = 3
	UNKNOWN_COUNTRY_CODE   = "ZZ"
	UNKNOWN_COUNTRY_CODE3  = "ZZZ"
)

var DEFAULT_COUNTRY_CODES_FOR_CONTINENTS = map[string]string{
	"Europe": "FR",
	"Asia":   "ID",
}

// FindGeoData finds the geolocation data for the given IP address.
func (g *OfflineGeocoder) FindGeoData(ctx context.Context, ipString string) (*Result, error) {
	var result Result
	var geoData GeoData
	ip := net.ParseIP(ipString)

	err := g.lookupIP(ip, geoData)
	if err != nil {
		return nil, err
	}

	countryCode := g.countryCodeFor(geoData)
	country, _ := g.findCachedCountry(ctx, countryCode)

	result.CountryCode = countryCode
	result.CountryCode3 = country.Alpha3Code
	result.UnknownCountry = countryCode == UNKNOWN_COUNTRY_CODE
	result.CountryID = country.ID
	result.CityName = geoData.City.Name
	result.RegionName = geoData.Subdivisions.MostSpecific.Name
	result.RegionCode = geoData.Subdivisions.MostSpecific.ISOCode
	result.Lat = geoData.Location.Latitude
	result.Lon = geoData.Location.Longitude
	result.Accuracy = geoData.Location.AccuracyRadius * 1000 // convert kilometers to meters
	result.ZipCode = geoData.Postal.Code
	result.IPService = MAX_MIND_PROVIDER_CODE

	return &result, nil
}

func (g *OfflineGeocoder) lookupIP(ip net.IP, geoData GeoData) error {
	err := g.MaxMindDB.Lookup(ip, &geoData)
	if err != nil {
		return err
	}
	return nil
}

func (g *OfflineGeocoder) countryCodeFor(geoData GeoData) string {
	if geoData.Country.ISOCode != "" {
		return geoData.Country.ISOCode
	}

	if code, ok := DEFAULT_COUNTRY_CODES_FOR_CONTINENTS[geoData.Continent.Name]; ok {
		return code
	}

	return UNKNOWN_COUNTRY_CODE
}

func (g *OfflineGeocoder) findCachedCountry(ctx context.Context, countryCode string) (*db.Country, error) {
	var dbCountry db.Country

	if err := g.DB.WithContext(ctx).Where("alpha_2_code = ?", countryCode).First(&dbCountry).Error; err != nil {
		return nil, err
	}

	return &dbCountry, nil
}
