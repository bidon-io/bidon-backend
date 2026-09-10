//go:build e2e

package e2e

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/dspsim"
)

const notifyTimeout = 5 * time.Second

// TestSDKFlow_Win drives the full happy path a real SDK walks: /v2/config,
// /v2/auction, /v2/win, /v2/show. It is the first test in the repo that
// chains more than one v2 handler, and the first to exercise the
// auction -> Redis -> win handoff (notification/store/auction_result_repo.go)
// against a real DSP.
func TestSDKFlow_Win(t *testing.T) {
	srv := newServer(t)
	f := newFixture(t, db.RewardedAdType, 0.1, true)

	var cfg configResponse
	resp := postJSON(t, srv, "/v2/config", newConfigRequest(t, f), &cfg)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, cfg.Init.Adapters, "adikteev")

	var auc auctionResponse
	resp = postJSON(t, srv, "/v2/auction/rewarded", newAuctionRequest(t, f, 0.1), &auc)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, auc.ConfigUID)
	require.Len(t, auc.AdUnits, 1, "expected exactly one RTB ad unit from dspsim")

	au := auc.AdUnits[0]
	require.Equal(t, "adikteev", au.DemandID)
	require.Equal(t, "RTB", au.BidType)
	require.NotNil(t, au.PriceFloor)
	require.Greater(t, *au.PriceFloor, 0.1, "the bid must be strictly above floor to have landed in ad_units")
	require.NotEmpty(t, au.Ext["payload"], "ad_unit.ext.payload should carry the DSP's adm")

	record := dspsimFindBid(t, f.app.PackageName.String)
	require.InDelta(t, *au.PriceFloor, record.Price, 0.0001)

	bid := newBid(auc.ConfigUID, auc.AuctionID, au)

	resp = postJSON(t, srv, "/v2/win/rewarded", newWinRequest(t, f, bid), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	win := waitForNotification(t, record.BidID, dspsim.NotificationWin, notifyTimeout)
	require.Empty(t, win.UnresolvedMacros, "nurl macros should all have been substituted")
	price, err := strconv.ParseFloat(win.Params["price"], 64)
	require.NoError(t, err)
	require.InDelta(t, *au.PriceFloor, price, 0.0001)

	resp = postJSON(t, srv, "/v2/show/rewarded", newShowRequest(t, f, bid), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	billing := waitForNotification(t, record.BidID, dspsim.NotificationBilling, notifyTimeout)
	require.Empty(t, billing.UnresolvedMacros, "burl macros should all have been substituted")
}

// TestSDKFlow_Loss drives an auction, then reports our bid as lost to an
// external (non-bidding) winner via /v2/loss, asserting dspsim recorded the
// lurl with the LossLostToHigherBid reason and the right first/second price.
func TestSDKFlow_Loss(t *testing.T) {
	srv := newServer(t)
	f := newFixture(t, db.RewardedAdType, 0.1, true)

	var auc auctionResponse
	resp := postJSON(t, srv, "/v2/auction/rewarded", newAuctionRequest(t, f, 0.1), &auc)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, auc.AdUnits, 1)

	au := auc.AdUnits[0]
	record := dspsimFindBid(t, f.app.PackageName.String)
	bid := newBid(auc.ConfigUID, auc.AuctionID, au)

	externalWinnerPrice := *au.PriceFloor + 5

	resp = postJSON(t, srv, "/v2/loss/rewarded", newLossRequest(t, f, bid, externalWinnerPrice), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	loss := waitForNotification(t, record.BidID, dspsim.NotificationLoss, notifyTimeout)
	require.Empty(t, loss.UnresolvedMacros, "lurl macros should all have been substituted")
	require.Equal(t, "102", loss.Params["loss"], "openrtb3.LossLostToHigherBid")

	price, err := strconv.ParseFloat(loss.Params["price"], 64)
	require.NoError(t, err)
	require.InDelta(t, externalWinnerPrice, price, 0.0001, "AUCTION_PRICE should be the external winner's price")

	mintowin, err := strconv.ParseFloat(loss.Params["mintowin"], 64)
	require.NoError(t, err)
	require.InDelta(t, *au.PriceFloor, mintowin, 0.0001, "AUCTION_MIN_TO_WIN should be our own (runner-up) price")
}

// TestSDKFlow_NoBid sets the auction floor above dspsim's DSPSIM_MAX_PRICE,
// forcing a deterministic 204 from the simulator, and asserts the SDK still
// gets a well-formed 200 with no RTB ad unit.
func TestSDKFlow_NoBid(t *testing.T) {
	srv := newServer(t)
	const floor = 30 // above the simulator's default DSPSIM_MAX_PRICE (25)
	f := newFixture(t, db.RewardedAdType, floor, true)

	var auc auctionResponse
	resp := postJSON(t, srv, "/v2/auction/rewarded", newAuctionRequest(t, f, floor), &auc)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, auc.AdUnits, "a floor above DSPSIM_MAX_PRICE should produce no RTB ad unit")
}
