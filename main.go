package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aint/cryptotokenlens/internal"
	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

// defaultExplorerAPIKey is the fallback when POLYGONSCAN_API_KEY and -api-key are empty.
// Prefer env/flag in shared repos so the key is not committed; rotate if this key leaks.
const defaultExplorerAPIKey = ""

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	apiKey := fs.String("api-key", getenv("POLYGONSCAN_API_KEY", defaultExplorerAPIKey), "Etherscan API v2 key (overrides POLYGONSCAN_API_KEY; default is built-in)")
	scanPause := fs.Duration("scan-pause", 400*time.Millisecond, "Extra pause between tokentx pages (free tier is often ~3 req/sec; client also spaces every call)")
	topHolders := fs.Int("top-holders", 15, "Show this many largest holders (0 = all)")
	_ = fs.Parse(os.Args[1:])

	client := polygonscan.NewClinet(*apiKey)
	token, err := internal.NewToken(internal.LaCasaEspanolaVilla9, client, *scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create token: %v\n", err)
		os.Exit(1)
	}

	printTokenInfo(token)

	internal.PrintHolders(token, *topHolders)

	dailySeries, err := internal.DailySeries(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build daily series: %v\n", err)
		os.Exit(1)
	}
	printDailySeries(dailySeries, token.Decimal)

	etas, err := internal.MovingAverageETA(dailySeries, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Calculate ETA: %v\n", err)
		os.Exit(1)
	}
	printETAs(etas)

	internal.WriteDailySeriesHTML(fmt.Sprintf("%s.html", token.Name), token, dailySeries, etas)
}

func printTokenInfo(token internal.Token) {
	fmt.Printf("Token %s, %s\n", token.Address, token.Name)
	fmt.Printf("total supply: %s\n", internal.FormatBigInt(token.TotalSupplyRaw, token.Decimal))
	fmt.Printf("%% bought (cumulative): %s%% (%s tokens)\n", internal.PercentOf(token.BoughtRaw, token.TotalSupplyRaw), internal.FormatBigInt(token.BoughtRaw, token.Decimal))
	fmt.Printf("Expected exit: %s\n", token.ETA.String())
	fmt.Println()
}

func printDailySeries(dailySeries []internal.DailyPoint, decimal uint8) {
	fmt.Printf("\nTimeline\n")
	fmt.Printf("%-12s %12s %12s\n", "day", "Δ", "% of supply")
	for _, p := range dailySeries {
		if p.Value.Sign() == 0 {
			continue
		}
		fmt.Printf("%-12s %12s %12.2f%%\n", p.Day.Format(time.DateOnly), internal.FormatBigInt(p.Value, decimal), p.CumPercent)
	}
}

func printETAs(etas []internal.ETA) {
	for _, eta := range etas {
		fmt.Printf("ETA: %s: %s tokens/day → ~%d calendar days → %s\n", eta.Window, eta.Rate, eta.Days, eta.Time.Format(time.DateOnly))
	}
}
