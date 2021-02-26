package main

import (
	"net"
	"strings"
)

type rpcServer interface {
	onConnect(commonName string) bool
	onDisconnect()
}

//Server -TODO-
type Server struct {
	servers []interface{}
}

func (s *Server) initServer(server interface{}, commonName string) {
	if server.(rpcServer).onConnect(commonName) {
		s.servers = append(s.servers, server)
	}
}

//OnConnect -TODO-
func (s *Server) OnConnect(commonName string, remoteAddress net.Addr) []interface{} {
	if commonName == cntlsPSCli { //
		sa := strings.Split(remoteAddress.String(), ":")

		if sa[0] != loopback {
			return s.servers
		}
	}

	s.initServer(&PayServ{}, commonName)
	s.initServer(&Utils{}, commonName)
	s.initServer(&Coins{}, commonName)
	s.initServer(&Wallets{}, commonName)
	s.initServer(&Check{}, commonName)
	s.initServer(&Corrections{}, commonName)
	s.initServer(&Debug{}, commonName)

	return s.servers
}

//OnDisconnect -TODO-
func (s *Server) OnDisconnect() {
	for _, server := range s.servers {
		server.(rpcServer).onDisconnect()
	}
}
