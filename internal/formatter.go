package internal

import (
	"math/big"
	"strings"
)

func FormatBigInt(raw *big.Int, decimals uint8) string {
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

func PercentOf(part, whole *big.Int) string {
	if whole.Sign() == 0 {
		return "0"
	}
	r := new(big.Rat).SetFrac(part, whole)
	r = new(big.Rat).Mul(r, big.NewRat(100, 1))
	return r.FloatString(4)
}