package main

import (
	"context"
	_ "context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Starting Binance Trading Bot...")
	loadEnv()

	client := binance.NewClient(
		os.Getenv("BINANCE_API_KEY"),
		os.Getenv("BINANCE_SECRET_KEY"),
	)

	symbol := "SOL/BRL"
	asset := "SOL"
	quantity := 0.001
	position := false

	for {
		klines, err := client.NewKlinesService().
			Symbol("SOLBRL").
			Interval("1h").
			Limit(1000).
			Do(context.Background())

		if err != nil {
			log.Println("Error fetching market data:", err)
			time.Sleep(60 * time.Second)
			continue
		}

		position = tradeStrategy(client, klines, symbol, asset, quantity, position)
		time.Sleep(60 * time.Second)
	}
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

func movingAverage(data []*binance.Kline, window int) float64 {
	if len(data) < window {
		return 0
	}
	sum := 0.0
	for i := len(data) - window; i < len(data); i++ {
		closePrice, err := strconv.ParseFloat(data[i].Close, 64)
		if err != nil {
			continue
		}
		sum += closePrice
	}
	return sum / float64(window)
}

func truncate(number float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Floor(number*factor) / factor
}

func tradeStrategy(client *binance.Client, data []*binance.Kline, symbol, asset string, quantity float64, position bool) bool {
	slowMean := movingAverage(data, 7)
	fastMean := movingAverage(data, 40)

	fmt.Printf("Last Fast Mean: %.4f, Last Slow Mean: %.4f\n", fastMean, slowMean)

	account, err := client.NewGetAccountService().Do(context.Background())
	if err != nil {
		log.Println("Error getting account info:", err)
		return position
	}

	var freeBalance float64
	for _, bal := range account.Balances {
		if bal.Asset == asset {
			freeBalance = toFloat(bal.Free)
			fmt.Printf("Free balance for %s: %.4f\n", asset, freeBalance)
			break
		}
	}

	if fastMean > slowMean && !position {
		fmt.Println("Placing buy order...")
		order, err := client.NewCreateOrderService().
			Symbol(symbol).
			Side(binance.SideTypeBuy).
			Type(binance.OrderTypeMarket).
			Quantity(fmt.Sprintf("%.3f", quantity)).
			Do(context.Background())
		fmt.Println(order)
		if err == nil {
			position = true
		} else {
			log.Println("Error placing buy order:", err)
		}
	} else if fastMean < slowMean && position {
		fmt.Println("Placing sell order...")
		order, err := client.NewCreateOrderService().
			Symbol(symbol).
			Side(binance.SideTypeSell).
			Type(binance.OrderTypeMarket).
			Quantity(fmt.Sprintf("%.3f", truncate(freeBalance, 3))).
			Do(context.Background())
		fmt.Println(order)
		if err == nil {
			position = false
		} else {
			log.Println("Error placing sell order:", err)
		}
	} else {
		fmt.Println("No trade action taken. Current position:", position)
	}

	return position
}
