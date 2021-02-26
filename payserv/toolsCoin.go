package main

var coins = map[string]int64{
	"ETH":         1,
	"EXP":         2,
	"UBQ":         3,
	"ZEN":         4,
	"ZCL":         5,
	"ZEC":         6,
	"KMD":         7,
	"MONA":        8,
	"HUSH":        9,
	"BTCZ":        10,
	"BTG":         11,
	"SIB":         12,
	"XMR":         13,
	"ETN":         14,
	"KRB":         15,
	"VTC":         16,
	"BTC":         17,
	"LTC":         19,
	"BCH":         20,
	"ETC":         21,
	"DASH":        22,
	"START":       23,
	"XMCC":        24,
	"XVG":         25,
	"LTZ":         26,
	"EMC2":        27,
	"MUSIC":       28,
	"XVG-BLAKE2S": 29,
	"XVG-SCRYPT":  30,
	"WHL":         31,
	"ETHTEST":     32,
	"RVN":         33,
}

//TagToID -
func TagToID(coinTag string) int64 {
	v, ok := coins[coinTag]

	if ok {
		return v
	}

	return 0
}

//IDToTag -
func IDToTag(coinID int64) string {
	for k, v := range coins {
		if v == coinID {
			return k
		}
	}

	return ""
}
