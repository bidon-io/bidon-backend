package geocoder

import (
	"context"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/db/dbtest"
	"github.com/oschwald/maxminddb-golang"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var testDB *db.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Prepare()

	os.Exit(m.Run())
}

func TestFindGeoData(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	mockMaxMindDB := new(maxminddb.Reader)

	// Create a test OfflineGeocoder instance
	geocoder := &OfflineGeocoder{
		maxMindDB: mockMaxMindDB,
		DB:        tx,
	}

	// Define test input and expected output
	ipString := "192.168.0.1"
	ip := net.ParseIP(ipString)
	geoData := GeoData{
		Country: struct {
			ISOCode string `maxminddb:"iso_code"`
		}{
			ISOCode: "US",
		},
		City: struct {
			Name string `maxminddb:"name"`
		}{
			Name: "New York",
		},
		Subdivisions: struct {
			MostSpecific struct {
				Name    string `maxminddb:"name"`
				ISOCode string `maxminddb:"iso_code"`
			} `maxminddb:"most_specific"`
		}{
			MostSpecific: struct {
				Name    string `maxminddb:"name"`
				ISOCode string `maxminddb:"iso_code"`
			}{
				Name:    "New York",
				ISOCode: "NY",
			},
		},
		Location: struct {
			Latitude       float64 `maxminddb:"latitude"`
			Longitude      float64 `maxminddb:"longitude"`
			AccuracyRadius int     `maxminddb:"accuracy_radius"`
		}{
			Latitude:       40.7128,
			Longitude:      -74.0060,
			AccuracyRadius: 10,
		},
		Postal: struct {
			Code string `maxminddb:"code"`
		}{
			Code: "10001",
		},
		Continent: struct {
			Name string `maxminddb:"name"`
		}{
			Name: "North America",
		},
	}
	expectedResult := &Result{
		CountryCode:    "US",
		CountryID:      1, // Assuming a specific country ID
		CountryCode3:   "",
		CityName:       "New York",
		RegionName:     "New York",
		RegionCode:     "NY",
		Lat:            40.7128,
		Lon:            -74.0060,
		Accuracy:       10000, // Converted from 10 kilometers to meters
		ZipCode:        "10001",
		IPService:      MAX_MIND_PROVIDER_CODE,
		UnknownCountry: false,
	}

	// Set up the mock expectations
	mockMaxMindDB.On("Lookup", ip, mock.AnythingOfType("*geocoder.GeoData")).Run(func(args mock.Arguments) {
		geoDataPtr := args.Get(1).(*GeoData)
		*geoDataPtr = geoData
	}).Return(nil)

	tx.On("WithContext", mock.Anything).Return(tx)
	mockDB.On("Find", mock.Anything, mock.AnythingOfType("map[string]any")).Run(func(args mock.Arguments) {
		out := args.Get(0).(*db.Country)
		out.ID = 1 // Assuming a specific country ID
	}).Return(nil)

	// Call the method under test
	result, err := geocoder.FindGeoData(context.Background(), ipString)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	// Verify the mock expectations
	mockMaxMindDB.AssertExpectations(t)
	mockDB.AssertExpectations(t)
}

func TestCountryCodeFor(t *testing.T) {
	geocoder := &OfflineGeocoder{}

	t.Run("Valid ISO code", func(t *testing.T) {
		geoData := GeoData{
			Country: struct {
				ISOCode string `maxminddb:"iso_code"`
			}{
				ISOCode: "US",
			},
		}
		expectedCode := "US"
		actualCode := geocoder.countryCodeFor(geoData)
		assert.Equal(t, expectedCode, actualCode)
	})

	t.Run("Default country code for continent", func(t *testing.T) {
		geoData := GeoData{
			Continent: struct {
				Name string `maxminddb:"name"`
			}{
				Name: "Europe",
			},
		}
		expectedCode := "FR"
		actualCode := geocoder.countryCodeFor(geoData)
		assert.Equal(t, expectedCode, actualCode)
	})

	t.Run("Unknown country code", func(t *testing.T) {
		geoData := GeoData{}
		expectedCode := "UNKNOWN"
		actualCode := geocoder.countryCodeFor(geoData)
		assert.Equal(t, expectedCode, actualCode)
	})
}
