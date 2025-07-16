package external

import "fmt"

// SecondsToInterval converts interval in seconds to Binance API interval format
func SecondsToInterval(seconds int) (string, error) {
	switch seconds {
	case 60:
		return "1m", nil
	case 180:
		return "3m", nil
	case 300:
		return "5m", nil
	case 900:
		return "15m", nil
	case 1800:
		return "30m", nil
	case 3600:
		return "1h", nil
	case 7200:
		return "2h", nil
	case 14400:
		return "4h", nil
	case 21600:
		return "6h", nil
	case 28800:
		return "8h", nil
	case 43200:
		return "12h", nil
	case 86400:
		return "1d", nil
	case 259200:
		return "3d", nil
	case 604800:
		return "1w", nil
	case 2592000:
		return "1M", nil
	default:
		return "", fmt.Errorf("unsupported interval: %d seconds. Supported intervals: 1m(60s), 3m(180s), 5m(300s), 15m(900s), 30m(1800s), 1h(3600s), 2h(7200s), 4h(14400s), 6h(21600s), 8h(28800s), 12h(43200s), 1d(86400s), 3d(259200s), 1w(604800s), 1M(2592000s)", seconds)
	}
}

// IntervalToSeconds converts Binance API interval format to seconds
func IntervalToSeconds(interval string) (int, error) {
	switch interval {
	case "1m":
		return 60, nil
	case "3m":
		return 180, nil
	case "5m":
		return 300, nil
	case "15m":
		return 900, nil
	case "30m":
		return 1800, nil
	case "1h":
		return 3600, nil
	case "2h":
		return 7200, nil
	case "4h":
		return 14400, nil
	case "6h":
		return 21600, nil
	case "8h":
		return 28800, nil
	case "12h":
		return 43200, nil
	case "1d":
		return 86400, nil
	case "3d":
		return 259200, nil
	case "1w":
		return 604800, nil
	case "1M":
		return 2592000, nil
	default:
		return 0, fmt.Errorf("unsupported interval: %s. Supported intervals: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M", interval)
	}
}

// GetSupportedIntervals returns a list of all supported intervals in seconds
func GetSupportedIntervals() []int {
	return []int{60, 180, 300, 900, 1800, 3600, 7200, 14400, 21600, 28800, 43200, 86400, 259200, 604800, 2592000}
}

// IsValidInterval checks if the given interval in seconds is supported
func IsValidInterval(seconds int) bool {
	_, err := SecondsToInterval(seconds)
	return err == nil
}