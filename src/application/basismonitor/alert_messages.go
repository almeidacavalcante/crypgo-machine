package basismonitor

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"crypgo-machine/src/domain/vo"
)

// DefaultLocale matches the language of the other Telegram notifications.
const DefaultLocale = "pt-BR"

// alertMessages is the text of one locale. Every entry is a format string;
// the arguments are listed next to each field. Messages use Telegram HTML.
type alertMessages struct {
	title      string // currency, threshold
	index      string // venue, currency, index price
	contract   string // instrument, days to expiry
	figures    string // premium, annualized, net annualized
	footer     string // formatted timestamp
	timeLayout string
}

var alertCatalog = map[string]alertMessages{
	"pt-BR": {
		title:      "📈 <b>BASE DE FUTUROS %s ACIMA DE %.1f%% a.a.</b>",
		index:      "%s · Índice %s: <b>%.2f USD</b>",
		contract:   "<code>%s</code> · %.0f dias",
		figures:    "Prêmio: <b>%.2f%%</b> · Anualizado: <b>%.2f%%</b> · Líquido de taxas: <b>%.2f%%</b>",
		footer:     "⏰ %s UTC · somente leitura, nenhuma ordem enviada",
		timeLayout: "02/01/2006 15:04",
	},
	"en": {
		title:      "📈 <b>%s FUTURES BASIS ABOVE %.1f%% p.a.</b>",
		index:      "%s · %s index: <b>%.2f USD</b>",
		contract:   "<code>%s</code> · %.0f days",
		figures:    "Premium: <b>%.2f%%</b> · Annualized: <b>%.2f%%</b> · Net of fees: <b>%.2f%%</b>",
		footer:     "⏰ %s UTC · read-only, no order was placed",
		timeLayout: "2006-01-02 15:04",
	},
}

// IsSupportedLocale reports whether alert text exists for the locale.
func IsSupportedLocale(locale string) bool {
	_, ok := alertCatalog[locale]
	return ok
}

// SupportedLocales lists the available locales in a stable order.
func SupportedLocales() []string {
	locales := make([]string, 0, len(alertCatalog))
	for locale := range alertCatalog {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

// formatAlert renders the alert for the contracts at or above the threshold.
// The locale must have been validated by Config.Validate.
func formatAlert(config Config, indexPrice float64, qualifying []vo.FuturesBasis, now time.Time) string {
	messages := alertCatalog[config.Locale]
	currency := html.EscapeString(config.Currency)

	var b strings.Builder
	b.WriteString(fmt.Sprintf(messages.title, currency, config.ThresholdPct))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf(messages.index, html.EscapeString(config.Venue), currency, indexPrice))
	b.WriteString("\n")

	for _, basis := range qualifying {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(messages.contract, html.EscapeString(basis.GetInstrument()), basis.GetDaysToExpiry()))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(messages.figures, basis.GetPremium(), basis.GetAnnualized(), basis.GetNetAnnualized()))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(messages.footer, now.UTC().Format(messages.timeLayout)))
	return b.String()
}
