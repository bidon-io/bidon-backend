package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
)

// ExecuteRTBOptions configures ExecuteRTBRequest.
// Network-specific URL selection, auth headers, query mutation, and post-response
// quirks stay in adapters via URL / Headers / PrepareURL / AfterDo.
type ExecuteRTBOptions struct {
	DemandID    adapter.Key
	URL         string
	TagID       string
	PlacementID string
	TimeoutURL  string
	ImpID       string
	Headers     http.Header
	PrepareURL  func(base string, request openrtb.BidRequest) (string, error)
	AfterDo     func(resp *http.Response, dr *DemandResponse)
}

// CountryFromRequest returns the device geo country (typically ISO alpha-3) from the bid request.
func CountryFromRequest(request openrtb.BidRequest) string {
	if request.Device != nil && request.Device.Geo != nil {
		return request.Device.Geo.Country
	}
	return ""
}

// ExecuteRTBRequest runs the common OpenRTB ExecuteRequest HTTP transport:
// marshal → POST JSON → read body → DemandResponse Status / RawResponse.
func ExecuteRTBRequest(
	ctx context.Context,
	client *http.Client,
	request openrtb.BidRequest,
	opts ExecuteRTBOptions,
) *DemandResponse {
	dr := &DemandResponse{
		DemandID:    opts.DemandID,
		RequestID:   request.ID,
		TagID:       opts.TagID,
		PlacementID: opts.PlacementID,
		TimeoutURL:  opts.TimeoutURL,
		ImpID:       opts.ImpID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		dr.Error = err
		return dr
	}
	dr.RawRequest = string(requestBody)

	url := opts.URL
	if opts.PrepareURL != nil {
		url, err = opts.PrepareURL(opts.URL, request)
		if err != nil {
			dr.Error = err
			return dr
		}
	}
	if url == "" {
		dr.Error = errors.New("endpoint URL is empty")
		return dr
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		dr.Error = err
		return dr
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for key, values := range opts.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

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

	if opts.AfterDo != nil {
		opts.AfterDo(httpResp, dr)
	}

	return dr
}
