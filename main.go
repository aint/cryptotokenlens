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

	x := internal.Tokens["La Casa Española Villa 6"]
	tokenAddr := &x

	client := polygonscan.NewClinet(*apiKey)
	totalSupply, err := client.GetTotalSupply(*tokenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explorer token metadata: %v\n", err)
		os.Exit(1)
	}
	txs, err := client.FetchAllTokenTx(*tokenAddr, 1000, *scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explorer API: %v\n", err)
		os.Exit(1)
	}
	decimal, err := internal.GetDecimal(txs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get decimal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Token %s\n", *tokenAddr)
	fmt.Printf("total supply: %s\n", internal.FormatBigInt(totalSupply, decimal))
	boughtAmount := internal.BoughtAmount(txs, *tokenAddr)
	if boughtAmount != nil {
		fmt.Printf("%% bought (cumulative): %s%% (%s tokens)\n", internal.PercentOf(boughtAmount, totalSupply), internal.FormatBigInt(boughtAmount, decimal))
	} else {
		fmt.Printf("%% bought (cumulative): n/a (no transfer window in fetched history)\n")
	}
	fmt.Println()

	internal.PrintHolders(txs, totalSupply, decimal, *topHolders)
	internal.PrintTimeline(txs, *tokenAddr, totalSupply, decimal)
	internal.PrintETA(txs, *tokenAddr, decimal, totalSupply, boughtAmount)
}
