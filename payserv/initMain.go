package main

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"log"
	"net"
	"net/rpc/jsonrpc"
	"strings"
	"time"

	"github.com/seagiv/common/coinapi"
	"github.com/seagiv/common/gutils"
)

func dbgwVersion(c *gutils.ClientConnection) error {
	var err error

	var reply gutils.ReplyVersion

	err = c.Call("Utils.Version", 0, &reply)
	if err != nil {
		return gutils.FormatErrorS("Utils.Version", "error %v", err)
	}

	if strings.Compare(reply.Version, buildInfo) != 0 {
		return gutils.FormatErrorS("", "payserv and dbgw version mismnatch must be %s", buildInfo)
	}

	if strings.Compare(reply.Project, projectInfo) != 0 {
		return gutils.FormatErrorS("", "payserv and dbgw project mismnatch must be %s", projectInfo)
	}

	return nil
}

func dbgwSetParams(c *gutils.ClientConnection) error {
	var err error

	var replyString string

	err = c.Call("Utils.SetParams", protectedParams, &replyString)
	if err != nil {
		return gutils.FormatErrorS("Utils.SetParams", "error %v", err)
	}

	return nil
}

func dbgwGetSendingCount(c *gutils.ClientConnection, coinID int64) (int64, error) {
	var err error

	var reply int64

	err = c.Call("Gateway.GetSendingCount", coinID, &reply)
	if err != nil {
		return 0, gutils.FormatErrorS("Utils.GetSendingCount", "error %v", err)
	}

	return reply, nil
}

const kmAddress = "127.0.0.1:8225" //local only, for security reasons, hardcoded
const kmRetryTimeout = 5           //seconds

var encryptedStorage *gutils.Storage

func getKey(fileName string) []byte {
	var stringKey string

	//keyS, _ := ioutil.ReadFile(fileName)

	//keyB, _ := hex.DecodeString(string(keyS))

	//return keyB

	gutils.RemoteLog.PutInfoN("requesting encryptedStorage key ...")

	for {
		conn, err := net.Dial("tcp", kmAddress)
		if err != nil {
			gutils.RemoteLog.PutInfoS("Dial", "connect to key manager failed, going to retry")

			time.Sleep(kmRetryTimeout * time.Second)

			continue
		}
		defer conn.Close()

		client := jsonrpc.NewClient(conn)

		err = client.Call("Manager.GetKey", 0, &stringKey)
		if err != nil {
			gutils.RemoteLog.PutErrorS("Manager.GetKey", "error %v, going to retry", err)
		} else {
			result, err := hex.DecodeString(stringKey)
			if err != nil {
				gutils.RemoteLog.PutFatalS("Decode", "error %v, going to retry", err)
			}

			return result
		}

		time.Sleep(kmRetryTimeout * time.Second)
	}
}

func initSecurity(fileName, testKeyFile string) {
	var err error

	storageKey := getKey(testKeyFile)

	gutils.RemoteLog.PutInfoS("encStorage", "opening [%s] ...", fileName)

	encryptedStorage, err = gutils.OpenStorage(fileName, storageKey, 0)
	if err != nil {
		gutils.RemoteLog.PutFatalS("OpenStorage", "%v", err)
	}

	gutils.RemoteLog.PutInfoS("encStorage", "open successfully.")

	buffer, ok := encryptedStorage.Get(sinParams)
	if !ok {
		gutils.RemoteLog.PutFatalS("encryptedStorage.Get(Params)", "%s not found in storage", sinParams)
	}

	protectedParams = string(buffer)

	certRoot, err := gutils.LoadCertificateFromStorage(encryptedStorage, sinRootCA)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadCertificateFromStorage(RootCA)", "%v", err)
	}

	if !certRoot.IsCA {
		gutils.RemoteLog.PutFatalS("certRoot", "root certificate isn't CA")
	}

	if bytes.Compare(certRoot.SubjectKeyId, certRoot.AuthorityKeyId) != 0 {
		gutils.RemoteLog.PutFatalS("certAuthority", "root certificate must be self signed")
	}

	//PayServ key/cert
	certPayServ, err = gutils.LoadCertificateFromStorage(encryptedStorage, sinPayServCert)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadCertificateFromStorag(PasyServ)", "%v", err)
	}

	keyPayServ, err = gutils.LoadPrivateKeyFromStorage(encryptedStorage, sinPayServKey)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadCertificateFromStorage(PayServ)", "%v", err)
	}

	//Admin key/cert
	certAdmin, err = gutils.LoadCertificateFromStorage(encryptedStorage, sinAdminCert)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadCertificateFromStorage(Admin)", "%v", err)
	}

	keyAdmin, err = gutils.LoadPrivateKeyFromStorage(encryptedStorage, sinAdminKey)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadPrivateKeyFromStorage(Admin)", "%v", err)
	}

	poolRoot = x509.NewCertPool()
	poolRoot.AddCert(certRoot)

	gutils.InfoTLS.Pair, err = gutils.LoadX509KeyPairFromStorage(encryptedStorage, sinTLSCert, sinTLSKey)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadX509KeyPairFromStorage(TLS)", "%v", err)
	}

	gutils.InfoTLS.ImCA, err = gutils.LoadCertificateFromStorage(encryptedStorage, sinTLSImCA)
	if err != nil {
		gutils.RemoteLog.PutFatalS("LoadCertificateFromStorage(imCA)", "%v", err)
	}
}

func accept(l *gutils.Listener, c net.Conn, s *Server) {
	err := l.AcceptConnect(c, s)
	if err != nil {
		gutils.RemoteLog.PutErrorN("%v", err)
	}
}

func initDB() (*gutils.ClientConnection, error) {
	client, err := gutils.InitClientTLS(
		addressDB,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
		cntlsDBGW,
		true,
	)
	if err != nil {
		gutils.RemoteLog.PutFatalS("InitClientTLS(1)", "%v", err)

		return nil, err
	}

	err = dbgwVersion(client)
	if err != nil {
		gutils.RemoteLog.PutFatalS("checkVersion", "%v", err)

		return nil, err
	}

	err = dbgwSetParams(client)
	if err != nil {
		gutils.RemoteLog.PutFatalS("setParams", "%v", err)

		return nil, err
	}

	client.Close() // reconnect required so dbgw will use params

	client, err = gutils.InitClientTLS(
		addressDB,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
		cntlsDBGW,
		true,
	)
	if err != nil {
		gutils.RemoteLog.PutFatalS("InitClientTLS(2)", "%v", err)

		return nil, err
	}

	sendingCount, err := dbgwGetSendingCount(client, TagToID(coinapi.CoinETH))
	if err != nil {
		gutils.RemoteLog.PutFatalS("getSendingCount", "%v", err)

		return nil, err
	}

	if sendingCount > 0 {
		gutils.RemoteLog.PutWarningN("%d transactions found in sending status for Ethereum!", sendingCount)
		gutils.RemoteLog.PutWarningN("Ethereum nonce may be out of sync!")
	}

	return client, nil
}

func initCoins(conf *config) error {
	var err error

	for k, v := range coinapi.Coins {
		v.Address, v.Key, _ = encryptedStorage.GetCoinInfo(k)
	}

	err = coinapi.InitEthereum(coinapi.CoinETH, conf.CoinETH, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinETH, "%v", err)
	}

	err = coinapi.InitEthereum(coinapi.CoinETC, conf.CoinETC, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinETC, "%v", err)
	}

	/*err = coinapi.InitBitcoin(coinapi.CoinBTC, conf.CoinBTC, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinBTC, "%v", err)
	}

	err = coinapi.InitBitcoin(coinapi.CoinBCH, conf.CoinBCH, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinBCH, "%v", err)
	}

	err = coinapi.InitBitcoin(coinapi.CoinZEC, conf.CoinZEC, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinZEC, "%v", err)
	}

	err = coinapi.InitBitcoin(coinapi.CoinRVN, conf.CoinRVN, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinRVN, "%v", err)
	}

	err = coinapi.InitBitcoin(coinapi.CoinDASH, conf.CoinDASH, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinDASH, "%v", err)
	}

	err = coinapi.InitBitcoin(coinapi.CoinMONA, conf.CoinMONA, testMode)
	if err != nil {
		gutils.RemoteLog.PutFatalS(coinapi.CoinMONA, "%v", err)
	}*/

	return err
}

func main() {
	var err error

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf := readConfig(configPath)

	gutils.RemoteLog = gutils.InitLog("payserv")

	gutils.RemoteLog.PutInfoN("STARTING [%s] (v%s) ...", projectInfo, buildInfo)

	txLocker = newLocker()

	initSecurity(conf.StorageFile, conf.TestKeyFile)

	sl, err := gutils.InitServerTLS(":"+conf.ListenPort,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
	)
	if err != nil {
		gutils.RemoteLog.PutFatalS("InitServerTLS", "%v", err)
	}

	addressDB = conf.AddressDB

	testMode = (conf.TestMode == "yes")

	if testMode {
		gutils.RemoteLog.PutWarningN("TEST MODE IS ON!")
	}

	err = initCoins(conf)
	if err != nil {
		return
	}

	gutils.InitCertCache(24*time.Hour, poolRoot)

	client, err := initDB()
	if err != nil {
		return
	}

	storageUserCP = NewUserCPStorage()
	err = storageUserCP.checkUserCPs(client)
	if err != nil {
		gutils.RemoteLog.PutFatalS("checkUserCPs", "%v", err)

		return
	}

	storageCoinCP = NewCoinCPStorage()
	err = storageCoinCP.checkCoinCPs(client)
	if err != nil {
		gutils.RemoteLog.PutFatalS("checkCoinCPs", "%v", err)

		return
	}

	client.Close()

	if testMode {
		enabledPayServ = true
	} else {
		gutils.RemoteLog.PutInfoN("Initialization complete. Manual PayServ activation required.")
	}

	for {
		conn, err := sl.Listener.Accept()
		if err != nil {
			gutils.RemoteLog.PutFatalS("Listener.Accept", "%v", err)
		}

		go accept(sl, conn, new(Server))
	}
}
