package mintegral

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type MintegralAdapter struct {
	SellerID    string
	AppID       string
	TagID       string
	PlacementID string
}

var _ adapters.BidderInterface = (*MintegralAdapter)(nil)

func (a *MintegralAdapter) banner(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	imp, _ := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	return imp
}

func (a *MintegralAdapter) interstitial(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	return adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{})
}

// rewarded stays custom: Instl=0 + Imp.Ext is_rewarded (not the shared dual/fullscreen shape).
func (a *MintegralAdapter) rewarded(auctionRequest *schema.AuctionRequest) *openrtb2.Imp {
	w, h := adapters.ResolveFullscreenSize(
		string(auctionRequest.Device.Type),
		auctionRequest.AdObject.IsPortrait(),
		true,
	)
	return &openrtb2.Imp{
		Instl: 0,
		Video: &openrtb2.Video{
			W:     w,
			H:     h,
			MIMEs: []string{"video/mp4"},
		},
		Ext: json.RawMessage(`{"is_rewarded": true}`),
	}
}

func (a *MintegralAdapter) BuildImpression(_ openrtb.BidRequest, auctionRequest *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	if a.TagID == "" {
		return nil, adapters.RTBRequestOptions{}, errors.New("TagID is empty")
	}

	var imp *openrtb2.Imp
	switch auctionRequest.AdObject.Type() {
	case ad.BannerType:
		imp = a.banner(auctionRequest)
	case ad.InterstitialType:
		imp = a.interstitial(auctionRequest)
	case ad.RewardedType:
		imp = a.rewarded(auctionRequest)
	default:
		return nil, adapters.RTBRequestOptions{}, errors.New("unknown impression type")
	}

	opts := adapters.RTBRequestOptions{
		TagID:       a.TagID,
		AppID:       a.AppID,
		PublisherID: a.SellerID,
	}
	if token, ok := auctionRequest.AdObject.Demands[adapter.MintegralKey]["token"].(string); ok {
		opts.BuyerUID = token
	}

	return imp, opts, nil
}

func (a *MintegralAdapter) EnrichOpenRTBRequest(request *openrtb.BidRequest, auctionRequest *schema.AuctionRequest) error {
	appExtStructure := &map[string]interface{}{}
	if auctionRequest.AdObject.IsPortrait() {
		(*appExtStructure)["orientation"] = 1
	} else {
		(*appExtStructure)["orientation"] = 2
	}
	raw, err := json.Marshal(appExtStructure)
	if err != nil {
		return err
	}
	request.App.Ext = raw
	return nil
}

func (a *MintegralAdapter) ExecuteRequest(ctx context.Context, client *http.Client, request openrtb.BidRequest) *adapters.DemandResponse {
	dr := &adapters.DemandResponse{
		DemandID:    adapter.MintegralKey,
		RequestID:   request.ID,
		TagID:       a.TagID,
		PlacementID: a.PlacementID,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	url := "http://hb.rayjump.com/bid"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("openrtb", "2.5")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		dr.Error = err
		return dr
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		dr.Error = err
		return dr
	}

	dr.RawResponse = string(respBody)
	dr.Status = httpResp.StatusCode

	return dr
}

// Builder builds a new instance of the Mintegral adapter for the given bidder with the given config.
func Builder(cfg adapter.ProcessedConfigsMap, client *http.Client) (*adapters.Bidder, error) {
	mCfg := cfg[adapter.MintegralKey]

	sellerID, ok := mCfg["seller_id"].(string)
	if !ok || sellerID == "" {
		return nil, fmt.Errorf("missing seller_id param for %s adapter", adapter.MintegralKey)
	}
	appID, ok := mCfg["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("missing app_id param for %s adapter", adapter.MintegralKey)
	}
	tagID, ok := mCfg["tag_id"].(string)
	if !ok {
		tagID = ""
	}
	placementID, ok := mCfg["placement_id"].(string)
	if !ok || placementID == "" {
		placementID = ""
	}

	adpt := &MintegralAdapter{
		SellerID:    sellerID,
		AppID:       appID,
		TagID:       tagID,
		PlacementID: placementID,
	}

	bidder := adapters.Bidder{
		Adapter: adpt,
		Client:  client,
	}

	return &bidder, nil
}
