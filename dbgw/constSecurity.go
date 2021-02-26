package main

const connectCorrections = "connectCorrections"
const connectGateway = "connectGateway"
const connectUtils = "connectUtils"

const cnPayServ = "PayServ"
const cntlsPSCli = "PSCli"

var allowedCNs = map[string][]string{
	connectGateway:     {cnPayServ},
	connectCorrections: {cnPayServ},
	connectUtils:       {cnPayServ, cntlsPSCli},
}
