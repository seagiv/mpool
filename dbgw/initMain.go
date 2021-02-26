package main

import (
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/seagiv/common/gutils"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := readConfig(configPath)

	gutils.RemoteLog = gutils.InitLog("dbgw")

	gutils.RemoteLog.PutInfoN("STARTING [%s] (v%s) ...", projectInfo, buildInfo)

	//dataSource = conf.DataSource

	tlsPair, err := gutils.LoadX509KeyPairFromFile(conf.Crypto.CertTLS, conf.Crypto.KeyTLS)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadX509KeyPairFromStorage(TLS)", "%v", err)
	}

	certImCA, err := gutils.LoadCertificateFromFile(conf.Crypto.ImCA)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadCertificateFromFile(imCA)", "%v", err)
	}

	sl, err := gutils.InitServerTLS(":"+
		conf.ListenPort,
		certImCA,
		tlsPair,
	)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := sl.Listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go sl.AcceptConnect(conn, new(Server))
	}
}
