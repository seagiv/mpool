package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
)

var testMode bool

const buildInfo = "1.0.25"
const projectInfo = "mpool"

const loopback = "127.0.0.1"

const limitSerial = 10000 // records per request for large requests (balances)

const configPath = "/etc/" + projectInfo + "/payserv.conf"

const ttCredit = 1
const ttFee = 8
const ttCreditRef = 12
const ttCorrection = 13

var certRoot *x509.Certificate

var certPayServ *x509.Certificate
var keyPayServ *rsa.PrivateKey

var certAdmin *x509.Certificate
var keyAdmin *rsa.PrivateKey

var certTLS *tls.Certificate

var poolRoot *x509.CertPool

var addressDB string

var protectedParams string
