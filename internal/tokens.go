package internal

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
	"errors"
	"strconv"

	"github.com/aint/cryptotokenlens/internal/polygonscan"
)

const (
	LaCasaEspañolaVilla4 = "La Casa Española Villa 4"
	LaCasaEspañolaVilla6 = "La Casa Española Villa 6"
	LaCasaEspañolaVilla8 = "La Casa Española Villa 8"
	LaCasaEspañolaVilla9 = "La Casa Española Villa 9"
	DukleyGlamping1 = "Dukley Glamping 1"
	RootsVilla1 = "Roots Villa 1"
	RootsVilla3 = "Roots Villa 3"
	RootsVilla4 = "Roots Villa 4"
	RootsVilla5 = "Roots Villa 5"
	CEMAGIUnit344 = "CEMAGI Unit 3.44"
	CEMAGIUnit346 = "CEMAGI Unit 3.46"
	CADECASVilla2 = "CASCADE Villa 2"
	CADECASVilla3 = "CASCADE Villa 3"
	BaliBalanceOceanVilla3 = "Bali Balance Ocean Villa 3"
	BaliBalanceOceanVilla4 = "Bali Balance Ocean Villa 4"
	BinginMagicStoryVilla3 = "Bingin Magic Story Villa 3"
	OasisRoyalCollection11a = "Oasis Royal Collection 11a"
	TaryanDragonJungleView = "Taryan Dragon Jungle View"
	AWWAHotelByRibasA16 = "AWWA Hotel by Ribas A16"
	AWWAHotelByRibasB14 = "AWWA Hotel by Ribas B14"
	AWWAHotelByRibasB22 = "AWWA Hotel by Ribas B22"
	OasisRoyalCollection18b = "Oasis Royal Collection 18b"
	MountainRetreatByDukley = "Mountain Retreat by Dukley"
	EcoverseSuite = "Ecoverse Suite"
)

type Token struct {
	Name           string
	Address        string
	Txs            []polygonscan.TokenTransfer
	TotalSupplyRaw *big.Int
	BoughtRaw      *big.Int
	RemainingRaw   *big.Int
	Decimal        uint8
	ETA            YearQuarter
}

type YearQuarter struct {
    Year    int
    Quarter int // 1..4
}

func (yq YearQuarter) String() string {
	return fmt.Sprintf("%d Q%d", yq.Year, yq.Quarter)
}

var tokens = map[string]Token{
	LaCasaEspañolaVilla4: {
		Name: LaCasaEspañolaVilla4,
		Address: "0x7b592d8bb722324f75af834c23e6ad2058b168e1",
		ETA: YearQuarter{ Year: 2026, Quarter: 4 },
	},
	LaCasaEspañolaVilla6: {
		Name: LaCasaEspañolaVilla6,
		Address: "0xdd36b686a5ff910b5074e3f5483135f19e49f02c",
		ETA: YearQuarter{ Year: 2026, Quarter: 4 },
	},
	LaCasaEspañolaVilla8: {
		Name: LaCasaEspañolaVilla8,
		Address: "0x223270bbbe4f6dac0dc3e57d985116bdc50616ee",
		ETA: YearQuarter{ Year: 2026, Quarter: 4 },
	},
	LaCasaEspañolaVilla9: {
		Name: LaCasaEspañolaVilla9,
		Address: "0x89ebdfaf79308871a24c6992232984b3c84af9a8",
		ETA: YearQuarter{ Year: 2026, Quarter: 4 },
	},
	DukleyGlamping1: {
		Name: DukleyGlamping1,
		Address: "0xad4f81d0f2f626a6ea29864f488604e6b5360e2a",
		ETA: YearQuarter{ Year: 2026, Quarter: 4 },
	},
	RootsVilla1: {
		Name: RootsVilla1,
		Address: "0xbde380b4cc582d440255ebd89ff1839dcfad5d7b",
		ETA: YearQuarter{ Year: 2026, Quarter: 3 },
	},
	RootsVilla3: {
		Name: RootsVilla3,
		Address: "0xc0a4b2e29bd44d3b798a02edc039711f03572739",
		ETA: YearQuarter{ Year: 2026, Quarter: 3 },
	},
	RootsVilla4: {
		Name: RootsVilla4,
		Address: "0xb2b9f922c0494dbf08636b1dbcf6fcba0878a605",
		ETA: YearQuarter{ Year: 2026, Quarter: 3 },
	},
	RootsVilla5: {
		Name: RootsVilla5,
		Address: "0x0ef68e86c3c9bc6187c69770053919e6b35991f6",
		ETA: YearQuarter{ Year: 2026, Quarter: 3 },
	},
	CEMAGIUnit344: {
		Name: CEMAGIUnit344,
		Address: "0x852b6995628b760c84bdd02bc143b48288d4dd3a",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	CEMAGIUnit346: {
		Name: CEMAGIUnit346,
		Address: "0x2b7dca2c2bafdb1dac0e01068091590fbe09e478",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	CADECASVilla2: {
		Name: CADECASVilla2,
		Address: "0x5e55b3e941f42732f1b941f2f673dc8811355e5e",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	CADECASVilla3: {
		Name: CADECASVilla3,
		Address: "0xd5551375d5ba01ddbcb38d20ac40671f26e6ada5",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	BaliBalanceOceanVilla3: {
		Name: BaliBalanceOceanVilla3,
		Address: "0x1e3cf2eeaa6d5973e2da6fe03600ba55870dd69b",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	BaliBalanceOceanVilla4: {
		Name: BaliBalanceOceanVilla4,
		Address: "0x17236ed296fbd00d3dfa016879833776dd207fd6",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	"Bingin Magic Story villa 3": {
		Name: BinginMagicStoryVilla3,
		Address: "0xe5f846592a58bcfce912bc6fc594649397b6f519",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
	OasisRoyalCollection11a: {
		Name: OasisRoyalCollection11a,
		Address: "0xa26f11748ed29b3fd62e1d8e231d277a0980fb12",
		ETA: YearQuarter{ Year: 2025, Quarter: 4 },
	},
	TaryanDragonJungleView: {
		Name: TaryanDragonJungleView,
		Address: "0x4bd4d7003a6ce76b9ad3ee364a29801c170b1ff5",
		ETA: YearQuarter{ Year: 2027, Quarter: 4 },
	},
	AWWAHotelByRibasB14: {
		Name: AWWAHotelByRibasB14,
		Address: "0x216301b87404a5839bf7b8b94c646c4eb96fec79",
		ETA: YearQuarter{ Year: 2025, Quarter: 2 },
	},
	AWWAHotelByRibasB22: {
		Name: AWWAHotelByRibasB22,
		Address: "0xe725a80f426a7d7f5734ba69ccec507251109d09",
		ETA: YearQuarter{ Year: 2025, Quarter: 2 },
	},
	AWWAHotelByRibasA16: {
		Name: AWWAHotelByRibasA16,
		Address: "0xdb8fc93a993e2ab0d9f7d520fd4e616cfb1d85fd",
		ETA: YearQuarter{ Year: 2025, Quarter: 2 },
	},
	OasisRoyalCollection18b: {
		Name: OasisRoyalCollection18b,
		Address: "0x1dac5a4a0e566fb2674a6b7e1cdaf2c07716eeed",
		ETA: YearQuarter{ Year: 2025, Quarter: 4 },
	},
	MountainRetreatByDukley: {
		Name: MountainRetreatByDukley,
		Address: "0x51343ee93059cbb11c4bf969a643e09117b3af6b",
		ETA: YearQuarter{ Year: 2024, Quarter: 4 },
	},
	EcoverseSuite: {
		Name: EcoverseSuite,
		Address: "0x30ed65e470be4f351abf5311769505e3f977deca",
		ETA: YearQuarter{ Year: 2026, Quarter: 2 },
	},
}

func NewToken(name, polygonScanAPIKey string, scanPause time.Duration) (Token, error) {
	token := tokens[name]
	client := polygonscan.NewClinet(polygonScanAPIKey)
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
