package deribit

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Trimmed copies of real responses from the public API (2026-11-20).
const (
	indexResponse = `{"jsonrpc":"2.0","result":{"estimated_delivery_price":61851.12,"index_price":61851.12},"testnet":false}`

	instrumentsResponse = `{"jsonrpc":"2.0","result":[
		{"instrument_name":"BTC-PERPETUAL","expiration_timestamp":32503708800000,"settlement_period":"perpetual","taker_commission":0.00035,"is_active":true,"kind":"future"},
		{"instrument_name":"BTC-4DEC26","expiration_timestamp":1796371200000,"settlement_period":"week","taker_commission":0.00035,"is_active":true,"kind":"future"},
		{"instrument_name":"BTC-26MAR27","expiration_timestamp":1806048000000,"settlement_period":"month","taker_commission":0.00035,"is_active":true,"kind":"future"},
		{"instrument_name":"BTC-25JUN27","expiration_timestamp":1813910400000,"settlement_period":"month","taker_commission":0.00035,"is_active":true,"kind":"future"},
		{"instrument_name":"BTC-24SEP27","expiration_timestamp":1821772800000,"settlement_period":"month","taker_commission":0.00035,"is_active":false,"kind":"future"}
	]}`

	summariesResponse = `{"jsonrpc":"2.0","result":[
		{"instrument_name":"BTC-PERPETUAL","bid_price":61836.5,"ask_price":61837.0},
		{"instrument_name":"BTC-4DEC26","bid_price":61915.0,"ask_price":61925.0},
		{"instrument_name":"BTC-26MAR27","bid_price":62970.0,"ask_price":62990.0},
		{"instrument_name":"BTC-25JUN27","bid_price":null,"ask_price":63897.5},
		{"instrument_name":"BTC-24SEP27","bid_price":64790.0,"ask_price":64830.0}
	]}`
)

func newTestServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func TestGetDatedFuturesSnapshot_ParsesTheCurve(t *testing.T) {
	var indexName, instrumentsQuery, summariesQuery string
	server := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/public/get_index_price": func(w http.ResponseWriter, r *http.Request) {
			indexName = r.URL.Query().Get("index_name")
			jsonHandler(indexResponse)(w, r)
		},
		"/api/v2/public/get_instruments": func(w http.ResponseWriter, r *http.Request) {
			instrumentsQuery = r.URL.RawQuery
			jsonHandler(instrumentsResponse)(w, r)
		},
		"/api/v2/public/get_book_summary_by_currency": func(w http.ResponseWriter, r *http.Request) {
			summariesQuery = r.URL.RawQuery
			jsonHandler(summariesResponse)(w, r)
		},
	})
	client := NewClientWithBaseURL(server.URL+"/", server.Client())

	snapshot, err := client.GetDatedFuturesSnapshot(" btc ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if indexName != "btc_usd" {
		t.Errorf("expected index btc_usd, got %q", indexName)
	}
	if instrumentsQuery != "currency=BTC&expired=false&kind=future" {
		t.Errorf("unexpected instruments query: %s", instrumentsQuery)
	}
	if summariesQuery != "currency=BTC&kind=future" {
		t.Errorf("unexpected book summary query: %s", summariesQuery)
	}

	if snapshot.IndexPrice != 61851.12 {
		t.Errorf("expected index price 61851.12, got %v", snapshot.IndexPrice)
	}
	if len(snapshot.Quotes) != 3 {
		t.Fatalf("expected 3 quotes (perpetual and inactive contracts left out), got %d", len(snapshot.Quotes))
	}

	weekly := snapshot.Quotes[0]
	if weekly.Instrument != "BTC-4DEC26" || weekly.BidPrice != 61915.0 {
		t.Errorf("unexpected weekly quote: %+v", weekly)
	}
	if math.Abs(weekly.FeeRate-0.00035) > 1e-12 {
		t.Errorf("weekly futures pay no delivery fee: expected fee rate 0.00035, got %v", weekly.FeeRate)
	}

	monthly := snapshot.Quotes[1]
	if monthly.Instrument != "BTC-26MAR27" || monthly.BidPrice != 62970.0 {
		t.Errorf("unexpected monthly quote: %+v", monthly)
	}
	if math.Abs(monthly.FeeRate-0.0006) > 1e-12 {
		t.Errorf("monthly futures pay taker plus delivery: expected fee rate 0.0006, got %v", monthly.FeeRate)
	}
	wantExpiry := time.Date(2027, 3, 26, 8, 0, 0, 0, time.UTC)
	if !monthly.Expiry.Equal(wantExpiry) {
		t.Errorf("expected expiry %v, got %v", wantExpiry, monthly.Expiry)
	}

	if noBid := snapshot.Quotes[2]; noBid.Instrument != "BTC-25JUN27" || noBid.BidPrice != 0 {
		t.Errorf("a contract without a bid must be reported with a zero bid, got %+v", noBid)
	}
}

func TestGetDatedFuturesSnapshot_RejectsEmptyCurrency(t *testing.T) {
	client := NewClientWithBaseURL("http://127.0.0.1:0", http.DefaultClient)
	if _, err := client.GetDatedFuturesSnapshot("  "); err == nil {
		t.Fatal("expected an error for an empty currency")
	}
}

func TestGetDatedFuturesSnapshot_ReportsFailures(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wants   string
	}{
		{
			name: "API error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"message":"Invalid params","code":-32602}}`))
			},
			wants: "API error -32602: Invalid params",
		},
		{
			name: "blocked without JSON",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("<html>forbidden</html>"))
			},
			wants: "unexpected HTTP status 403",
		},
		{
			name:    "invalid JSON",
			handler: jsonHandler("not json"),
			wants:   "invalid JSON response",
		},
		{
			name:    "missing result",
			handler: jsonHandler(`{"jsonrpc":"2.0"}`),
			wants:   "no result",
		},
		{
			name:    "zero index price",
			handler: jsonHandler(`{"jsonrpc":"2.0","result":{"index_price":0}}`),
			wants:   "invalid BTC index price",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t, map[string]http.HandlerFunc{
				"/api/v2/public/get_index_price": tc.handler,
			})
			client := NewClientWithBaseURL(server.URL, server.Client())

			_, err := client.GetDatedFuturesSnapshot("BTC")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wants) {
				t.Errorf("expected error to contain %q, got %v", tc.wants, err)
			}
		})
	}
}
