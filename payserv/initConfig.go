package main

import (
	"fmt"
	"os"

	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/toml"
)

type config struct {
	ListenPort string
	AddressDB  string
	TestMode   string

	StorageFile string
	TestKeyFile string

	CoinETH  gutils.CoinConfig
	CoinETC  gutils.CoinConfig
	CoinBTC  gutils.CoinConfig
	CoinBCH  gutils.CoinConfig
	CoinZEC  gutils.CoinConfig
	CoinRVN  gutils.CoinConfig
	CoinDASH gutils.CoinConfig
	CoinMONA gutils.CoinConfig
}

func readConfig(filename string) *config {
	var conf config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		fmt.Printf("ERROR: %v", err)

		os.Exit(1)
	}
	return &conf
}
