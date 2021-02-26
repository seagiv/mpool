package main

import (
	"log"

	"github.com/seagiv/foreign/toml"
)

type config struct {
	AddressSVR  string
	AddressDBGW string
	Crypto      struct {
		ImCA    string
		CertTLS string
		KeyTLS  string
	}
	Local struct {
		DataSource string
		CertPath   string
		KeyPath    string
	}
}

func readConfig(filename string) *config {
	var conf config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		log.Fatal(err)
	}
	return &conf
}
