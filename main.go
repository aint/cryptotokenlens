package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

const espaniolaVila9Token = "0x89EbdFaf79308871A24c6992232984b3C84af9A8"
const montenegroToken = "0xaD4f81D0F2f626A6EA29864F488604e6b5360e2a"

// defaultExplorerAPIKey is the fallback when POLYGONSCAN_API_KEY and -api-key are empty.
// Prefer env/flag in shared repos so the key is not committed; rotate if this key leaks.
const defaultExplorerAPIKey = "???"

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

type tokenMeta struct {
	decimals    uint8
	totalSupply *big.Int
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
	topHolders := fs.Int("top-holders", 20, "Show this many largest holders (0 = all)")
	_ = fs.Parse(os.Args[1:])

	if err := validateTokenAddr(*tokenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	client := &polygonscan.Client{APIKey: apiKeyTrim, ChainID: *chainID}

	dec, sup, err := client.ERC20MetaFromExplorer(tokenLower)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explorer token metadata: %v\n", err)
		os.Exit(1)
	}
	meta := tokenMeta{decimals: dec, totalSupply: sup}

	txs, err := client.FetchAllTokenTx(*tokenAddr, 1000, *scanPause)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explorer API: %v\n", err)
		os.Exit(1)
	}
	boughtDays, metricFromContract := buildBoughtTimeline(txs, tokenLower)

	fmt.Printf("Token %s\n", tokenLower)
	fmt.Printf("totalSupply: %s (%s raw)\n", formatUnits(meta.totalSupply, meta.decimals), meta.totalSupply.String())
	fmt.Printf("tokentx rows (fetched): %d\n", len(txs))
	if len(boughtDays) > 0 {
		lastCum := boughtDays[len(boughtDays)-1].cum
		fmt.Printf("%% bought (cumulative): %s%% (%s tokens)\n", pctBoughtDisplay(lastCum, meta.totalSupply), formatTokenAmountExact(lastCum, meta.decimals))
	} else {
		fmt.Printf("%% bought (cumulative): n/a (no transfer window in fetched history)\n")
	}
	fmt.Println()

	balances, err := polygonscan.ReplayBalances(txs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay balances: %v\n", err)
		os.Exit(1)
	}

	printHolders(os.Stdout, balances, meta.totalSupply, meta.decimals, *topHolders)
	firstTS, lastTS, ok := transferTimeBounds(txs)
	if len(boughtDays) > 0 {
		printBoughtTimeline(os.Stdout, boughtDays, meta.totalSupply, meta.decimals, firstTS, lastTS, ok, metricFromContract)
	}
}

const zeroAddr0x = "0x0000000000000000000000000000000000000000"

type boughtDayRow struct {
	dayUTC string
	cum    *big.Int // cumulative “bought” amount through end of this UTC day (see buildBoughtTimeline)
}

func transferTimeBounds(txs []polygonscan.TokenTransfer) (first, last int64, ok bool) {
	for _, t := range txs {
		if t.TxreceiptStatus == "0" {
			continue
		}
		ts, err := strconv.ParseInt(strings.TrimSpace(t.TimeStamp), 10, 64)
		if err != nil {
			continue
		}
		if !ok || ts < first {
			first = ts
		}
		if !ok || ts > last {
			last = ts
		}
		ok = true
	}
	return first, last, ok
}

// buildBoughtTimeline lists each UTC day from first→last transfer.
// If any transfer has from == tokenAddr (supply leaving the contract), %bought uses that cumulative total
// (typical sale/inventory). Otherwise it falls back to cumulative mint-from-0x0.
func buildBoughtTimeline(txs []polygonscan.TokenTransfer, tokenAddr string) ([]boughtDayRow, bool) {
	tokenAddr = strings.ToLower(strings.TrimSpace(tokenAddr))
	mintByDay := make(map[string]*big.Int)
	outByDay := make(map[string]*big.Int)
	var minDay, maxDay time.Time
	gotBound := false

	for _, t := range txs {
		if t.TxreceiptStatus == "0" {
			continue
		}
		tsi, err := strconv.ParseInt(strings.TrimSpace(t.TimeStamp), 10, 64)
		if err != nil {
			continue
		}
		tm := time.Unix(tsi, 0).UTC()
		dayStr := tm.Format("2006-01-02")
		day0, err := time.ParseInLocation("2006-01-02", dayStr, time.UTC)
		if err != nil {
			continue
		}
		if !gotBound {
			minDay, maxDay = day0, day0
			gotBound = true
		} else {
			if day0.Before(minDay) {
				minDay = day0
			}
			if day0.After(maxDay) {
				maxDay = day0
			}
		}

		v, ok := new(big.Int).SetString(t.Value, 10)
		if !ok {
			continue
		}

		from := strings.ToLower(strings.TrimSpace(t.From))
		if from == tokenAddr {
			if outByDay[dayStr] == nil {
				outByDay[dayStr] = big.NewInt(0)
			}
			outByDay[dayStr].Add(outByDay[dayStr], v)
		}
		if from == zeroAddr0x {
			if mintByDay[dayStr] == nil {
				mintByDay[dayStr] = big.NewInt(0)
			}
			mintByDay[dayStr].Add(mintByDay[dayStr], v)
		}
	}

	if !gotBound {
		return nil, false
	}

	totalOut := big.NewInt(0)
	for _, v := range outByDay {
		totalOut.Add(totalOut, v)
	}

	series := outByDay
	fromContract := totalOut.Sign() > 0
	if !fromContract {
		series = mintByDay
	}

	cum := big.NewInt(0)
	var rows []boughtDayRow
	for d := minDay; !d.After(maxDay); d = d.AddDate(0, 0, 1) {
		ds := d.Format("2006-01-02")
		if m := series[ds]; m != nil {
			cum = new(big.Int).Add(cum, m)
		}
		rows = append(rows, boughtDayRow{dayUTC: ds, cum: new(big.Int).Set(cum)})
	}
	return rows, fromContract
}

func printHolders(w io.Writer, balances map[string]*big.Int, totalSupply *big.Int, decimals uint8, top int) {
	type row struct {
		addr string
		bal  *big.Int
	}
	var rows []row
	for a, b := range balances {
		if b.Sign() <= 0 {
			continue
		}
		rows = append(rows, row{a, new(big.Int).Set(b)})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].bal.Cmp(rows[j].bal) > 0
	})
	if top > 0 && len(rows) > top {
		rows = rows[:top]
	}
	fmt.Fprintf(w, "\nHolders (positive balance): showing %d of %d with non-zero balance\n", len(rows), countNonZero(balances))
	fmt.Fprintf(w, "%-44s %32s %14s\n", "address", "balance", "% of supply")
	for _, r := range rows {
		fmt.Fprintf(w, "%s %32s %13s%%\n", r.addr, formatUnits(r.bal, decimals), pctOf(r.bal, totalSupply))
	}
}

func countNonZero(balances map[string]*big.Int) int {
	n := 0
	for _, b := range balances {
		if b.Sign() > 0 {
			n++
		}
	}
	return n
}

func printBoughtTimeline(w io.Writer, rows []boughtDayRow, totalSupply *big.Int, decimals uint8, firstTS, lastTS int64, boundsOK bool, fromContract bool) {
	if fromContract {
		fmt.Fprintf(w, "\nBought timeline (UTC): cumulative %% of supply that left the token contract (Transfer from token address).\n")
	} else {
		fmt.Fprintf(w, "\nBought timeline (UTC): cumulative %% of supply minted from 0x0 (no transfers from token contract in data).\n")
	}
	fmt.Fprintf(w, "TokensPurchased = cumulative token count (same basis as %%bought), exact from raw amount / 10^decimals.\n")
	fmt.Fprintf(w, "Δ = TokensPurchased minus previous calendar day (tokens added that UTC day). Only days with Δ ≠ 0 are listed.\n\n")
	hdr := "%-12s %12s %40s %16s\n"
	fmt.Fprintf(w, hdr, "day", "%bought", "TokensPurchased", "Δ")
	line := "%-12s %11s%% %40s %16s\n"
	var lastCum *big.Int
	var prevCum *big.Int
	for _, r := range rows {
		lastCum = r.cum
		oldPrev := prevCum
		var base *big.Int
		if oldPrev == nil {
			base = big.NewInt(0)
		} else {
			base = oldPrev
		}
		d := new(big.Int).Sub(r.cum, base)
		prevCum = r.cum
		if d.Sign() == 0 {
			continue
		}
		tok := formatTokenAmountExact(r.cum, decimals)
		delta := formatDayDelta(r.cum, oldPrev, decimals)
		fmt.Fprintf(w, line, r.dayUTC, pctBoughtDisplay(r.cum, totalSupply), tok, delta)
	}

	eta := linearETA100(firstTS, lastTS, lastCum, totalSupply, boundsOK)
	fmt.Fprintf(w, "\nLinear ETA to 100%% bought (constant rate from first→last tx time): %s\n", eta)
}

func pctBoughtDisplay(cum, supply *big.Int) string {
	if supply == nil || supply.Sign() == 0 {
		return "0"
	}
	if cum.Cmp(supply) >= 0 {
		return "100.0000"
	}
	return pctOf(cum, supply)
}

// linearETA100 assumes progress 0%% at firstTS and cum/supply at lastTS; extrapolates to 100%%.
func linearETA100(firstTS, lastTS int64, cumMint, totalSupply *big.Int, boundsOK bool) string {
	if !boundsOK || totalSupply == nil || totalSupply.Sign() == 0 {
		return "n/a"
	}
	if cumMint.Sign() <= 0 {
		return "n/a (0%% bought in the chosen metric)"
	}
	if cumMint.Cmp(totalSupply) >= 0 {
		return "n/a (already ≥100%% — projection not needed)"
	}
	p := new(big.Rat).SetFrac(cumMint, totalSupply)
	elapsed := lastTS - firstTS
	if elapsed <= 0 {
		t := time.Unix(lastTS, 0).UTC().Format(time.RFC3339)
		return fmt.Sprintf("%s (no elapsed window between first/last tx)", t)
	}
	one := big.NewRat(1, 1)
	remFrac := new(big.Rat).Sub(one, p)
	secRem := new(big.Rat).Mul(big.NewRat(elapsed, 1), remFrac)
	secRem.Quo(secRem, p)
	sf, _ := secRem.Float64()
	if math.IsInf(sf, 0) || math.IsNaN(sf) || sf < 0 {
		return "n/a"
	}
	eta := time.Unix(lastTS, 0).UTC().Add(time.Duration(sf * float64(time.Second)))
	return eta.Format(time.RFC3339)
}

func formatUnits(i *big.Int, dec uint8) string {
	if i == nil {
		return "0"
	}
	return formatTokenAmountExact(i, dec)
}

// formatDayDelta formats cum − prevCum in human token units; prevCum nil means previous day had cumulative 0.
func formatDayDelta(cum, prevCum *big.Int, decimals uint8) string {
	if cum == nil {
		return "0"
	}
	var base *big.Int
	if prevCum == nil {
		base = big.NewInt(0)
	} else {
		base = prevCum
	}
	d := new(big.Int).Sub(cum, base)
	if d.Sign() == 0 {
		return "0"
	}
	if d.Sign() > 0 {
		return "+" + formatTokenAmountExact(d, decimals)
	}
	return "-" + formatTokenAmountExact(new(big.Int).Neg(d), decimals)
}

// formatTokenAmountExact formats raw ERC-20 units as a decimal token amount using only integer math (no float drift).
func formatTokenAmountExact(raw *big.Int, decimals uint8) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}
	if decimals == 0 {
		return raw.String()
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	ip := new(big.Int).Quo(raw, denom)
	fp := new(big.Int).Mod(new(big.Int).Set(raw), denom)
	if fp.Sign() == 0 {
		return ip.String()
	}
	frac := fp.Text(10)
	for len(frac) < int(decimals) {
		frac = "0" + frac
	}
	frac = strings.TrimRight(frac, "0")
	return ip.String() + "." + frac
}

func pctOf(part, whole *big.Int) string {
	if whole.Sign() == 0 {
		return "0"
	}
	r := new(big.Rat).SetFrac(part, whole)
	r = new(big.Rat).Mul(r, big.NewRat(100, 1))
	return r.FloatString(4)
}
