package geocoder

import (
	"context"
	"database/sql"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/db/dbtest"
	"github.com/oschwald/maxminddb-golang"
	"net"
	"os"
	"testing"
)

var testDB *db.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Prepare()

	os.Exit(m.Run())
}

func TestFindGeoData(t *testing.T) {
	// Create a test database connection
	tx := testDB.Begin()
	defer tx.Rollback()

	countries := []db.Country{
		{
			Alpha2Code: "ZZ",
			HumanName: sql.NullString{
				String: "Unknown",
				Valid:  true,
			},
		},
	}
	if err := tx.Create(&countries).Error; err != nil {
		t.Fatalf("Error creating configs: %v", err)
	}

	maxMindDB, err := maxminddb.Open("../../../public/GeoIP2-City-Test.mmdb")
	if err != nil {
		panic(err)
	}

	// Create an instance of OfflineGeocoder using the test database connection
	geocoder := &OfflineGeocoder{
		MaxMindDB: maxMindDB,
		DB:        tx,
	}

	// Define the test case input
	ipString := "127.0.0.1"

	// Call the method being tested
	result, err := geocoder.FindGeoData(context.Background(), ipString)

	// Perform assertions on the result and error
	// Example assertions:
	if err != nil {
		t.Errorf("FindGeoData returned an unexpected error: %v", err)
	}

	if result == nil {
		t.Error("FindGeoData returned nil result, expected non-nil result")
	}

	// Add more specific assertions based on the expected behavior of FindGeoData
}

func TestLookupIP(t *testing.T) {
	// Create a mock instance of maxminddb.Reader

	maxMindDB, err := maxminddb.Open("../../../public/GeoIP2-City-Test.mmdb")
	if err != nil {
		panic(err)
	}

	// Create an instance of OfflineGeocoder using the mock dependencies
	geocoder := &OfflineGeocoder{
		MaxMindDB: maxMindDB,
	}

	// Define the test case input
	ip := net.ParseIP("127.0.0.1")
	var geoData GeoData

	// Call the method being tested
	err = geocoder.lookupIP(ip, geoData)

	// Perform assertions on the error
	// Example assertions:
	if err != nil {
		t.Errorf("lookupIP returned an unexpected error: %v", err)
	}

	// Add more specific assertions based on the expected behavior of lookupIP
}

func TestCountryCodeFor(t *testing.T) {
	// Create an instance of OfflineGeocoder
	geocoder := &OfflineGeocoder{}

	// Define the test case input
	var geoData GeoData

	// Call the method being tested
	countryCode := geocoder.countryCodeFor(geoData)

	// Perform assertions on the countryCode
	// Example assertions:
	expectedCode := "ZZ"
	if countryCode != expectedCode {
		t.Errorf("countryCodeFor returned %s, expected %s", countryCode, expectedCode)
	}

	// Add more specific assertions based on the expected behavior of countryCodeFor
}

func TestFindCachedCountry(t *testing.T) {
	// Create a test database connection
	tx := testDB.Begin()
	defer tx.Rollback()

	countries := []db.Country{
		{
			Alpha2Code: "US",
			HumanName: sql.NullString{
				String: "United States",
				Valid:  true,
			},
		},
	}
	if err := tx.Create(&countries).Error; err != nil {
		t.Fatalf("Error creating configs: %v", err)
	}

	// Create an instance of OfflineGeocoder using the test database connection
	geocoder := &OfflineGeocoder{
		DB: tx,
	}

	// Define the test case input
	ctx := context.Background()
	countryCode := "US"

	// Call the method being tested
	country, err := geocoder.findCachedCountry(ctx, countryCode)

	// Perform assertions on the country and error
	// Example assertions:
	if err != nil {
		t.Errorf("findCachedCountry returned an unexpected error: %v", err)
	}

	if country == nil {
		t.Error("findCachedCountry returned nil country, expected non-nil country")
	}

	// Add more specific assertions based on the expected behavior of findCachedCountry
}
