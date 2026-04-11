package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aint/cryptotokenlens/internal"
	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

const (
	laCasaEspanolaV4Token = "0x7b592d8bb722324f75af834c23e6ad2058b168e1"
	laCasaEspanolaV6Token = "0xdd36b686a5ff910b5074e3f5483135f19e49f02c"
	laCasaEspanolaV8Token = "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee"
	laCasaEspanolaV9Token = "0x89EbdFaf79308871A24c6992232984b3C84af9A8"
	dukleyGlamping1Token  = "0xaD4f81D0F2f626A6EA29864F488604e6b5360e2a"
	rootsV1Token          = "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b"
	rootsV3Token          = "0xc0a4b2e29bd44d3b798a02edc039711f03572739"
	rootsV4Token          = "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605"
	rootsV5Token          = "0x0ef68e86c3c9bc6187c69770053919e6b35991f6"
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

func validateTokenAddr(addr string) error {
	if len(addr) != 42 || !strings.HasPrefix(addr, "0x") || strings.ToLower(addr) != addr {
		return errors.New("token address must be 42 lowercase chars: 0x + 40 hex digits")
	}
	if _, err := hex.DecodeString(addr[2:]); err != nil {
		return errors.New("token address must be hex")
	}
	return nil
}

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	apiKey := fs.String("api-key", getenv("POLYGONSCAN_API_KEY", defaultExplorerAPIKey), "Etherscan API v2 key (overrides POLYGONSCAN_API_KEY; default is built-in)")
	tokenAddr := fs.String("token", rootsV1Token, "ERC-20 contract address")
	scanPause := fs.Duration("scan-pause", 400*time.Millisecond, "Extra pause between tokentx pages (free tier is often ~3 req/sec; client also spaces every call)")
	topHolders := fs.Int("top-holders", 15, "Show this many largest holders (0 = all)")
	_ = fs.Parse(os.Args[1:])

	if err := validateTokenAddr(*tokenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

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
}
