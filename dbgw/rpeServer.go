package main

import (
	"net"
)

//const creditAllB = "2,3,5,7" //1, 8, 12, 13
//const creditUserB = "2,5,7"  //1, 12, 13

const creditAllU = "1,8,12,13"
const creditUserU = "1,12,13"

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
	s.initServer(&Gateway{}, commonName)
	s.initServer(&Corrections{}, commonName)
	s.initServer(&Utils{}, commonName)

	return s.servers
}

//OnDisconnect -TODO-
func (s *Server) OnDisconnect() {
	for _, server := range s.servers {
		server.(rpcServer).onDisconnect()
	}
}
