package external

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance/v2"
)

// BinanceClientInterface defines the interface for Binance client operations
type BinanceClientInterface interface {
	NewKlinesService() KlinesServiceInterface
	NewCreateOrderService() CreateOrderServiceInterface
	NewGetAccountService() GetAccountServiceInterface
}

// KlinesServiceInterface defines the interface for klines operations
type KlinesServiceInterface interface {
	Symbol(string) KlinesServiceInterface
	Interval(string) KlinesServiceInterface
	StartTime(int64) KlinesServiceInterface
	EndTime(int64) KlinesServiceInterface
	Limit(int) KlinesServiceInterface
	Do(context.Context) ([]*binance.Kline, error)
}

// CreateOrderServiceInterface defines the interface for order operations
type CreateOrderServiceInterface interface {
	Symbol(string) CreateOrderServiceInterface
	Side(binance.SideType) CreateOrderServiceInterface
	Type(binance.OrderType) CreateOrderServiceInterface
	Quantity(string) CreateOrderServiceInterface
	Do(context.Context) (*binance.CreateOrderResponse, error)
}

// GetAccountServiceInterface defines the interface for account operations
type GetAccountServiceInterface interface {
	Do(context.Context) (*binance.Account, error)
}

// BinanceClientFake simulates Binance API responses for testing
type BinanceClientFake struct {
	predefinedKlines []*binance.Kline
	shouldFailKlines bool
	shouldFailOrder  bool
	orderCounter     int
}

func NewBinanceClientFake() *BinanceClientFake {
	return &BinanceClientFake{
		predefinedKlines: make([]*binance.Kline, 0),
		orderCounter:     1,
	}
}

// SetPredefinedKlines sets the klines that will be returned by the fake client
func (f *BinanceClientFake) SetPredefinedKlines(klines []*binance.Kline) {
	f.predefinedKlines = klines
}

// SetShouldFailKlines makes the klines service fail
func (f *BinanceClientFake) SetShouldFailKlines(shouldFail bool) {
	f.shouldFailKlines = shouldFail
}

// SetShouldFailOrder makes the order service fail
func (f *BinanceClientFake) SetShouldFailOrder(shouldFail bool) {
	f.shouldFailOrder = shouldFail
}

// NewKlinesService returns a fake klines service
func (f *BinanceClientFake) NewKlinesService() KlinesServiceInterface {
	return &FakeKlinesService{
		client: f,
	}
}

// NewCreateOrderService returns a fake order service
func (f *BinanceClientFake) NewCreateOrderService() CreateOrderServiceInterface {
	return &FakeCreateOrderService{
		client: f,
	}
}

// NewGetAccountService returns a fake account service
func (f *BinanceClientFake) NewGetAccountService() GetAccountServiceInterface {
	return &FakeGetAccountService{
		client: f,
	}
}

// FakeKlinesService simulates the Binance KlinesService
type FakeKlinesService struct {
	client    *BinanceClientFake
	symbol    string
	interval  string
	startTime int64
	endTime   int64
	limit     int
}

func (s *FakeKlinesService) Symbol(symbol string) KlinesServiceInterface {
	s.symbol = symbol
	return s
}

func (s *FakeKlinesService) Interval(interval string) KlinesServiceInterface {
	s.interval = interval
	return s
}

func (s *FakeKlinesService) StartTime(startTime int64) KlinesServiceInterface {
	s.startTime = startTime
	return s
}

func (s *FakeKlinesService) EndTime(endTime int64) KlinesServiceInterface {
	s.endTime = endTime
	return s
}

func (s *FakeKlinesService) Limit(limit int) KlinesServiceInterface {
	s.limit = limit
	return s
}

func (s *FakeKlinesService) Do(ctx context.Context) ([]*binance.Kline, error) {
	if s.client.shouldFailKlines {
		return nil, fmt.Errorf("simulated klines error")
	}

	// Return predefined klines or generate default ones
	if len(s.client.predefinedKlines) > 0 {
		return s.client.predefinedKlines, nil
	}

	// Generate default test klines
	return s.generateDefaultKlines(), nil
}

func (s *FakeKlinesService) generateDefaultKlines() []*binance.Kline {
	limit := s.limit
	if limit == 0 {
		limit = 50 // Default limit
	}
	
	klines := make([]*binance.Kline, limit)
	basePrice := 100.0

	for i := 0; i < limit; i++ {
		price := basePrice + float64(i)*0.5
		klines[i] = &binance.Kline{
			Open:      fmt.Sprintf("%.2f", price),
			Close:     fmt.Sprintf("%.2f", price+0.1),
			High:      fmt.Sprintf("%.2f", price+0.3),
			Low:       fmt.Sprintf("%.2f", price-0.2),
			Volume:    "100.0",
			CloseTime: 1640995200000 + int64(i*3600000), // Each hour
		}
	}

	return klines
}

// FakeCreateOrderService simulates the Binance CreateOrderService
type FakeCreateOrderService struct {
	client    *BinanceClientFake
	symbol    string
	side      binance.SideType
	orderType binance.OrderType
	quantity  string
}

func (s *FakeCreateOrderService) Symbol(symbol string) CreateOrderServiceInterface {
	s.symbol = symbol
	return s
}

func (s *FakeCreateOrderService) Side(side binance.SideType) CreateOrderServiceInterface {
	s.side = side
	return s
}

func (s *FakeCreateOrderService) Type(orderType binance.OrderType) CreateOrderServiceInterface {
	s.orderType = orderType
	return s
}

func (s *FakeCreateOrderService) Quantity(quantity string) CreateOrderServiceInterface {
	s.quantity = quantity
	return s
}

func (s *FakeCreateOrderService) Do(ctx context.Context) (*binance.CreateOrderResponse, error) {
	if s.client.shouldFailOrder {
		return nil, fmt.Errorf("simulated order error")
	}

	s.client.orderCounter++

	return &binance.CreateOrderResponse{
		Symbol:  s.symbol,
		OrderID: int64(s.client.orderCounter),
		Side:    s.side,
		Status:  binance.OrderStatusTypeFilled,
		Type:    s.orderType,
	}, nil
}

// FakeGetAccountService simulates the Binance GetAccountService
type FakeGetAccountService struct {
	client *BinanceClientFake
}

func (s *FakeGetAccountService) Do(ctx context.Context) (*binance.Account, error) {
	return &binance.Account{
		Balances: []binance.Balance{
			{
				Asset: "BTC",
				Free:  "1.0",
			},
			{
				Asset: "ETH",
				Free:  "10.0",
			},
		},
	}, nil
}

// CreateWhipsawKlines creates klines that will cause whipsaw signals
func CreateWhipsawKlines() []*binance.Kline {
	baseKlines := []*binance.Kline{}
	
	// Generate 40 base klines with stable trend
	basePrice := 800.0
	for i := 0; i < 40; i++ {
		price := basePrice + float64(i)*0.5
		baseKlines = append(baseKlines, &binance.Kline{
			Open:      fmt.Sprintf("%.1f", price),
			Close:     fmt.Sprintf("%.1f", price+0.3),
			High:      fmt.Sprintf("%.1f", price+0.5),
			Low:       fmt.Sprintf("%.1f", price-0.2),
			Volume:    "100",
			CloseTime: 1640995200000 + int64(i*3600000),
		})
	}
	
	// Add whipsaw klines at the end
	baseKlines = append(baseKlines, []*binance.Kline{
		{Open: "830.0", Close: "831.0", High: "832.0", Low: "829.0", Volume: "100", CloseTime: 1641139200000},
		{Open: "831.0", Close: "832.0", High: "833.0", Low: "830.0", Volume: "100", CloseTime: 1641142800000},
		{Open: "832.0", Close: "833.0", High: "834.0", Low: "831.0", Volume: "100", CloseTime: 1641146400000},
		{Open: "833.0", Close: "835.0", High: "836.0", Low: "832.0", Volume: "100", CloseTime: 1641150000000},
		{Open: "835.0", Close: "834.5", High: "836.0", Low: "833.0", Volume: "100", CloseTime: 1641153600000},
		{Open: "834.5", Close: "834.0", High: "835.0", Low: "833.0", Volume: "100", CloseTime: 1641157200000},
		{Open: "834.0", Close: "833.5", High: "834.5", Low: "832.0", Volume: "100", CloseTime: 1641160800000},
	}...)
	
	return baseKlines
}

// CreateStrongTrendKlines creates klines with strong trend and sufficient spread
func CreateStrongTrendKlines() []*binance.Kline {
	baseKlines := []*binance.Kline{}
	
	// Generate 40 base klines with moderate trend
	basePrice := 700.0
	for i := 0; i < 40; i++ {
		price := basePrice + float64(i)*1.5 // Gradual uptrend
		baseKlines = append(baseKlines, &binance.Kline{
			Open:      fmt.Sprintf("%.1f", price),
			Close:     fmt.Sprintf("%.1f", price+1.2),
			High:      fmt.Sprintf("%.1f", price+1.5),
			Low:       fmt.Sprintf("%.1f", price-0.5),
			Volume:    "100",
			CloseTime: 1640995200000 + int64(i*3600000),
		})
	}
	
	// Add strong trend klines at the end
	baseKlines = append(baseKlines, []*binance.Kline{
		{Open: "760.0", Close: "770.0", High: "771.0", Low: "759.0", Volume: "100", CloseTime: 1641139200000},
		{Open: "770.0", Close: "780.0", High: "781.0", Low: "769.0", Volume: "100", CloseTime: 1641142800000},
		{Open: "780.0", Close: "790.0", High: "791.0", Low: "779.0", Volume: "100", CloseTime: 1641146400000},
		{Open: "790.0", Close: "800.0", High: "801.0", Low: "789.0", Volume: "100", CloseTime: 1641150000000},
		{Open: "800.0", Close: "810.0", High: "811.0", Low: "799.0", Volume: "100", CloseTime: 1641153600000},
		{Open: "810.0", Close: "820.0", High: "821.0", Low: "809.0", Volume: "100", CloseTime: 1641157200000},
		{Open: "820.0", Close: "830.0", High: "831.0", Low: "819.0", Volume: "100", CloseTime: 1641160800000},
	}...)
	
	return baseKlines
}

// BinanceClientWrapper wraps the real binance.Client to implement BinanceClientInterface
type BinanceClientWrapper struct {
	client *binance.Client
}

func NewBinanceClientWrapper(client *binance.Client) *BinanceClientWrapper {
	return &BinanceClientWrapper{
		client: client,
	}
}

func (w *BinanceClientWrapper) NewKlinesService() KlinesServiceInterface {
	return &RealKlinesService{
		service: w.client.NewKlinesService(),
	}
}

func (w *BinanceClientWrapper) NewCreateOrderService() CreateOrderServiceInterface {
	return &RealCreateOrderService{
		service: w.client.NewCreateOrderService(),
	}
}

func (w *BinanceClientWrapper) NewGetAccountService() GetAccountServiceInterface {
	return &RealGetAccountService{
		service: w.client.NewGetAccountService(),
	}
}

// RealKlinesService wraps the real binance klines service
type RealKlinesService struct {
	service *binance.KlinesService
}

func (s *RealKlinesService) Symbol(symbol string) KlinesServiceInterface {
	s.service = s.service.Symbol(symbol)
	return s
}

func (s *RealKlinesService) Interval(interval string) KlinesServiceInterface {
	s.service = s.service.Interval(interval)
	return s
}

func (s *RealKlinesService) StartTime(startTime int64) KlinesServiceInterface {
	s.service = s.service.StartTime(startTime)
	return s
}

func (s *RealKlinesService) EndTime(endTime int64) KlinesServiceInterface {
	s.service = s.service.EndTime(endTime)
	return s
}

func (s *RealKlinesService) Limit(limit int) KlinesServiceInterface {
	s.service = s.service.Limit(limit)
	return s
}

func (s *RealKlinesService) Do(ctx context.Context) ([]*binance.Kline, error) {
	return s.service.Do(ctx)
}

// RealCreateOrderService wraps the real binance order service
type RealCreateOrderService struct {
	service *binance.CreateOrderService
}

func (s *RealCreateOrderService) Symbol(symbol string) CreateOrderServiceInterface {
	s.service = s.service.Symbol(symbol)
	return s
}

func (s *RealCreateOrderService) Side(side binance.SideType) CreateOrderServiceInterface {
	s.service = s.service.Side(side)
	return s
}

func (s *RealCreateOrderService) Type(orderType binance.OrderType) CreateOrderServiceInterface {
	s.service = s.service.Type(orderType)
	return s
}

func (s *RealCreateOrderService) Quantity(quantity string) CreateOrderServiceInterface {
	s.service = s.service.Quantity(quantity)
	return s
}

func (s *RealCreateOrderService) Do(ctx context.Context) (*binance.CreateOrderResponse, error) {
	return s.service.Do(ctx)
}

// RealGetAccountService wraps the real binance account service
type RealGetAccountService struct {
	service *binance.GetAccountService
}

func (s *RealGetAccountService) Do(ctx context.Context) (*binance.Account, error) {
	return s.service.Do(ctx)
}
