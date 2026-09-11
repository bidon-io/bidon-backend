//go:build e2e

package e2e

import (
	"database/sql"
	"testing"

	"github.com/lib/pq"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/db/dbtest"
)

// fixture is a complete, committed Adikteev bidding setup for one ad type:
// an app, a demand source account pointing at dspsim, an enabled demand
// profile, one bidding line item, and a v2 auction configuration. It is the
// minimal row set that lets a real /v2/auction request reach dspsim and get
// the resulting bid back into the response — see the trace in BAC-69.
type fixture struct {
	app      db.App
	account  db.DemandSourceAccount
	profile  db.AppDemandProfile
	lineItem db.LineItem
	config   db.AuctionConfiguration
}

// newFixture commits a fresh app + Adikteev bidding setup for adType, with
// the auction configuration's pricefloor set to floor, and tells dspsim to
// reload its catalog so the fixture is visible immediately.
func newFixture(t *testing.T, adType db.AdType, floor float64, externalWinNotifications bool) fixture {
	t.Helper()

	app := dbtest.CreateApp(t, testDB, func(a *db.App) {
		a.PlatformID = db.AndroidPlatformID
	})

	account := dbtest.CreateDemandSourceAccount(t, testDB, func(a *db.DemandSourceAccount) {
		a.DemandSource = adikteevSource
		a.Extra = []byte(`{"endpoint": "` + dspsimURL + `/openrtb/bid"}`)
	})

	profile := dbtest.CreateAppDemandProfile(t, testDB, func(p *db.AppDemandProfile) {
		p.App = app
		p.Account = account
		p.Data = dbtest.ValidAppDemandProfileData(t, adapter.AdikteevKey, app.ID)
	})

	lineItem := dbtest.CreateLineItem(t, testDB, func(li *db.LineItem) {
		li.App = app
		li.Account = account
		li.AdType = adType
		li.IsBidding = sql.NullBool{Bool: true, Valid: true}
	})

	externalWin := externalWinNotifications
	config := dbtest.CreateAuctionConfiguration(t, testDB, func(c *db.AuctionConfiguration) {
		c.App = app
		c.AdType = adType
		c.Bidding = pq.StringArray{string(adapter.AdikteevKey)}
		c.AdUnitIds = pq.Int64Array{lineItem.ID}
		c.Pricefloor = floor
		c.ExternalWinNotifications = &externalWin
	})

	dspsimReload(t)

	return fixture{
		app:      app,
		account:  account,
		profile:  profile,
		lineItem: lineItem,
		config:   config,
	}
}
