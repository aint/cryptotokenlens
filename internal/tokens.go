package internal

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

type Token struct {
	TokenDetails
	Txs            []polygonscan.TokenTransfer
	TotalSupplyRaw *big.Int
	BoughtRaw      *big.Int
	RemainingRaw   *big.Int
	Decimal        uint8
}

type TokenDetails struct {
	Name           string
	Address        string
	ETA            YearQuarter
}

type YearQuarter struct {
    Year    int
    Quarter int // 1..4
}

func (yq YearQuarter) String() string {
	return fmt.Sprintf("%d Q%d", yq.Year, yq.Quarter)
}

func NewToken(tokenDetails TokenDetails, client *polygonscan.Client, scanPause time.Duration) (Token, error) {
	token := Token{
		TokenDetails: tokenDetails,
	}

	var err error
	token.TotalSupplyRaw, err = client.GetTotalSupply(token.Address)
	if err != nil {
		return Token{}, fmt.Errorf("get total supply: %v", err)
	}

	token.Txs, err = client.FetchAllTokenTx(token.Address, 1000, scanPause)
	if err != nil {
		return Token{}, fmt.Errorf("fetch all token tx: %v", err)
	}
	if len(token.Txs) == 0 {
		return Token{}, fmt.Errorf("no transactions found")
	}

	token.Decimal, err = token.decimal()
	if err != nil {
		return Token{}, fmt.Errorf("get decimal: %v", err)
	}

	token.BoughtRaw = token.boughtRaw()
	token.RemainingRaw = new(big.Int).Sub(token.TotalSupplyRaw, token.BoughtRaw)

	return token, nil
}

func (token *Token) boughtRaw() *big.Int {
	boughtAmount := big.NewInt(0)
	for _, t := range token.Txs {
		v, ok := new(big.Int).SetString(t.Value, 10)
		if !ok {
			fmt.Fprintf(os.Stderr, "parse value %q\n", t.Value)
			continue
		}

		from := strings.ToLower(t.From)
		if from == token.Address {
			boughtAmount.Add(boughtAmount, v)
		}
	}

	return boughtAmount
}

func (token *Token) decimal() (uint8, error) {
	decimalStr := strings.TrimSpace(token.Txs[0].TokenDecimal)
	if decimalStr == "" {
		return 0, errors.New("decimal missing")
	}
	decimal, err := strconv.ParseUint(decimalStr, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("parse decimal %q: %w", decimalStr, err)
	}
	return uint8(decimal), nil
}

var LaCasaEspanolaVilla4 = TokenDetails{
	Name: "La Casa Española Villa 4",
	Address: "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
	ETA: YearQuarter{ Year: 2026, Quarter: 4 },
}
var LaCasaEspanolaVilla6 = TokenDetails{
	Name: "La Casa Española Villa 6",
	Address: "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
	ETA: YearQuarter{ Year: 2026, Quarter: 4 },
}
var LaCasaEspanolaVilla8 = TokenDetails{
	Name: "La Casa Española Villa 8",
	Address: "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
	ETA: YearQuarter{ Year: 2026, Quarter: 4 },
}
var LaCasaEspanolaVilla9 = TokenDetails{
	Name: "La Casa Española Villa 9",
	Address: "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
	ETA: YearQuarter{ Year: 2026, Quarter: 4 },
}

var LaCasaEspanolaVillas = []TokenDetails{
	LaCasaEspanolaVilla4,
	LaCasaEspanolaVilla6,
	LaCasaEspanolaVilla8,
	LaCasaEspanolaVilla9,
}

var RootsVilla1 = TokenDetails{
	Name: "Roots Villa 1",
	Address: "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
	ETA: YearQuarter{ Year: 2026, Quarter: 3 },
}
var RootsVilla3 = TokenDetails{
	Name: "Roots Villa 3",
	Address: "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
	ETA: YearQuarter{ Year: 2026, Quarter: 3 },
}
var RootsVilla4 = TokenDetails{
	Name: "Roots Villa 4",
	Address: "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
	ETA: YearQuarter{ Year: 2026, Quarter: 3 },
}
var RootsVilla5 = TokenDetails{
	Name: "Roots Villa 5",
	Address: "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
	ETA: YearQuarter{ Year: 2026, Quarter: 3 },
}

var RootsVillas = []TokenDetails{
	RootsVilla1,
	RootsVilla3,
	RootsVilla4,
	RootsVilla5,
}

var DukleyGlamping1 = TokenDetails{
	Name: "Dukley Glamping 1",
	Address: "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
	ETA: YearQuarter{ Year: 2026, Quarter: 4 },
}
var MountainRetreatByDukley = TokenDetails{
	Name: "Mountain Retreat by Dukley",
	Address: "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
	ETA: YearQuarter{ Year: 2024, Quarter: 4 },
}

var Dukley = []TokenDetails{
	DukleyGlamping1,
	MountainRetreatByDukley,
}

var CemagiUnit344 = TokenDetails{
	Name: "CEMAGI Unit 3.44",
	Address: "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}
var CemagiUnit346 = TokenDetails{
	Name: "CEMAGI Unit 3.46",
	Address: "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}

var CemagiUnits = []TokenDetails{
	CemagiUnit344,
	CemagiUnit346,
}

var CadecasVilla2 = TokenDetails{
	Name: "CASCADE Villa 2",
	Address: "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}
var CadecasVilla3 = TokenDetails{
	Name: "CASCADE Villa 3",
	Address: "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}

var CadecasVillas = []TokenDetails{
	CadecasVilla2,
	CadecasVilla3,
}

var BaliBalanceOceanVilla3 = TokenDetails{
	Name: "Bali Balance Ocean Villa 3",
	Address: "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}
var BaliBalanceOceanVilla4 = TokenDetails{
	Name: "Bali Balance Ocean Villa 4",
	Address: "0x17236ed296fbd00d3dfa016879833776dd207fd6",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}

var BaliBalanceOceanVillas = []TokenDetails{
	BaliBalanceOceanVilla3,
	BaliBalanceOceanVilla4,
}

var BinginMagicStoryVilla3 = TokenDetails{
	Name: "Bingin Magic Story Villa 3",
	Address: "0xe5f846592a58bcfce912bc6fc594649397b6f519",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}

var BinginMagicStoryVillas = []TokenDetails{
	BinginMagicStoryVilla3,
}

var OasisRoyalCollection11a = TokenDetails{
	Name: "Oasis Royal Collection 11a",
	Address: "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
	ETA: YearQuarter{ Year: 2025, Quarter: 4 },
}
var OasisRoyalCollection18b = TokenDetails{
	Name: "Oasis Royal Collection 18b",
	Address: "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
	ETA: YearQuarter{ Year: 2025, Quarter: 4 },
}

var OasisRoyalCollection = []TokenDetails{
	OasisRoyalCollection11a,
	OasisRoyalCollection18b,
}

var TaryanDragonJungleView = TokenDetails{
	Name: "Taryan Dragon Jungle View",
	Address: "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
	ETA: YearQuarter{ Year: 2027, Quarter: 4 },
}

var TaryanDragonJungleViews = []TokenDetails{
	TaryanDragonJungleView,
}

var AWWAHotelByRibasB14 = TokenDetails{
	Name: "AWWA Hotel by Ribas B14",
	Address: "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
	ETA: YearQuarter{ Year: 2025, Quarter: 2 },
}
var AWWAHotelByRibasB22 = TokenDetails{
	Name: "AWWA Hotel by Ribas B22",
	Address: "0xe725a80f426a7d7f5734ba69ccec507251109d09",
	ETA: YearQuarter{ Year: 2025, Quarter: 2 },
}
var AWWAHotelByRibasA16 = TokenDetails{
	Name: "AWWA Hotel by Ribas A16",
	Address: "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
	ETA: YearQuarter{ Year: 2025, Quarter: 2 },
}

var AWWAHotelByRibas = []TokenDetails{
	AWWAHotelByRibasB14,
	AWWAHotelByRibasB22,
	AWWAHotelByRibasA16,
}

var EcoverseSuite = TokenDetails{
	Name: "Ecoverse Suite",
	Address: "0x30ed65e470be4f351abf5311769505e3f977deca",
	ETA: YearQuarter{ Year: 2026, Quarter: 2 },
}

var EcoverseSuites = []TokenDetails{
	EcoverseSuite,
}

var AllTokenDetails = [][]TokenDetails{
	LaCasaEspanolaVillas,
	RootsVillas,
	Dukley,
	CemagiUnits,
	CadecasVillas,
	BaliBalanceOceanVillas,
	BinginMagicStoryVillas,
	OasisRoyalCollection,
	TaryanDragonJungleViews,
	AWWAHotelByRibas,
	EcoverseSuites,
}