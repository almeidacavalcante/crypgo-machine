// Package deribit reads public market data from the Deribit exchange.
// It uses only unauthenticated endpoints: no API key is needed.
package deribit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"crypgo-machine/src/application/basismonitor"
)

const (
	// VenueName is the display name used in alerts.
	VenueName = "Deribit"

	DefaultBaseURL = "https://www.deribit.com"

	userAgent      = "CrypGo-Basis-Monitor/1.0"
	requestTimeout = 20 * time.Second
	maxBodyBytes   = 4 << 20

	settlementPerpetual = "perpetual"
	settlementMonth     = "month"

	// monthlyDeliveryFeeRate is charged at expiry on monthly and quarterly
	// futures; weekly and daily futures are exempt. The API does not expose
	// it. Source: Deribit "Fees" help article, checked on 2026-11-20.
	monthlyDeliveryFeeRate = 0.00025
)

// Client is a read-only Deribit market data client.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return NewClientWithBaseURL(DefaultBaseURL, &http.Client{Timeout: requestTimeout})
}

// NewClientWithBaseURL builds a client against another host, such as the
// Deribit testnet or a test server.
func NewClientWithBaseURL(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type instrument struct {
	Name             string  `json:"instrument_name"`
	ExpirationMillis int64   `json:"expiration_timestamp"`
	SettlementPeriod string  `json:"settlement_period"`
	TakerCommission  float64 `json:"taker_commission"`
	IsActive         bool    `json:"is_active"`
}

type bookSummary struct {
	Name     string   `json:"instrument_name"`
	BidPrice *float64 `json:"bid_price"`
}

type indexPrice struct {
	IndexPrice float64 `json:"index_price"`
}

// GetDatedFuturesSnapshot returns the spot index and every active dated
// future settled in the given currency (for example "BTC"). Perpetuals are
// left out because they never converge to the index at a known date.
func (c *Client) GetDatedFuturesSnapshot(currency string) (basismonitor.FuturesMarketSnapshot, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return basismonitor.FuturesMarketSnapshot{}, errors.New("currency cannot be empty")
	}

	var index indexPrice
	indexQuery := url.Values{"index_name": {strings.ToLower(currency) + "_usd"}}
	if err := c.get("public/get_index_price", indexQuery, &index); err != nil {
		return basismonitor.FuturesMarketSnapshot{}, err
	}
	if index.IndexPrice <= 0 {
		return basismonitor.FuturesMarketSnapshot{}, fmt.Errorf("deribit returned an invalid %s index price: %v", currency, index.IndexPrice)
	}

	var instruments []instrument
	instrumentsQuery := url.Values{"currency": {currency}, "kind": {"future"}, "expired": {"false"}}
	if err := c.get("public/get_instruments", instrumentsQuery, &instruments); err != nil {
		return basismonitor.FuturesMarketSnapshot{}, err
	}

	var summaries []bookSummary
	summariesQuery := url.Values{"currency": {currency}, "kind": {"future"}}
	if err := c.get("public/get_book_summary_by_currency", summariesQuery, &summaries); err != nil {
		return basismonitor.FuturesMarketSnapshot{}, err
	}

	bids := make(map[string]float64, len(summaries))
	for _, summary := range summaries {
		if summary.BidPrice != nil {
			bids[summary.Name] = *summary.BidPrice
		}
	}

	snapshot := basismonitor.FuturesMarketSnapshot{IndexPrice: index.IndexPrice}
	for _, inst := range instruments {
		if !inst.IsActive || inst.SettlementPeriod == settlementPerpetual {
			continue
		}
		snapshot.Quotes = append(snapshot.Quotes, basismonitor.FuturesQuote{
			Instrument: inst.Name,
			Expiry:     time.UnixMilli(inst.ExpirationMillis).UTC(),
			BidPrice:   bids[inst.Name],
			FeeRate:    inst.TakerCommission + deliveryFeeRate(inst.SettlementPeriod),
		})
	}
	return snapshot, nil
}

func deliveryFeeRate(settlementPeriod string) float64 {
	if settlementPeriod == settlementMonth {
		return monthlyDeliveryFeeRate
	}
	return 0
}

// get calls a public JSON-RPC method over HTTP GET and decodes its result.
func (c *Client) get(method string, query url.Values, out any) error {
	endpoint := fmt.Sprintf("%s/api/v2/%s?%s", c.baseURL, method, query.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("deribit %s: failed to create request: %w", method, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deribit %s: request failed: %w", method, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("deribit %s: failed to read response: %w", method, err)
	}

	var envelope rpcEnvelope
	decodeErr := json.Unmarshal(body, &envelope)
	if decodeErr == nil && envelope.Error != nil {
		return fmt.Errorf("deribit %s: API error %d: %s", method, envelope.Error.Code, envelope.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deribit %s: unexpected HTTP status %d", method, resp.StatusCode)
	}
	if decodeErr != nil {
		return fmt.Errorf("deribit %s: invalid JSON response: %w", method, decodeErr)
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("deribit %s: response has no result", method)
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("deribit %s: unexpected result shape: %w", method, err)
	}
	return nil
}
