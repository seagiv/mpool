package main

import (
	"log"
	"os"

	"github.com/seagiv/foreign/toml"
)

type config struct {
	ListenPort string
	ClientCN   string
	DataSource string
	Crypto     struct {
		ImCA    string
		CertTLS string
		KeyTLS  string
	}
}

func readConfig(filename string) *config {
	var conf config
	if _, err := toml.DecodeFile(filename, &conf); err != nil {
		log.Printf("ERROR: %v", err)

		os.Exit(1)
	}
	return &conf
}
