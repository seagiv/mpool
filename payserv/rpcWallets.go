package main

import (
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/mpool/mutils"
)

//Wallets --TODO--
type Wallets struct {
	сlientCN string

	c *gutils.ClientConnection
}

func (w *Wallets) onConnect(commonName string) bool {
	w.сlientCN = commonName

	return gutils.IsIn(commonName, allowedCNs[connectWallets])
}

func (w *Wallets) onDisconnect() {
}

func (w *Wallets) cleanUp() error {
	if w.c != nil {
		w.c.Close()
	}

	return nil
}

func (w *Wallets) initStage1(id int64) error {
	var err error

	gutils.RemoteLog.PutInfoN("CALL(%d) [%s]", id, w.сlientCN)

	w.c, err = gutils.InitClientTLS(
		addressDB,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
		cntlsDBGW,
		true,
	)
	if err != nil {
		return gutils.FormatErrorS("InitClient", "(%s) error %v", addressDB, err)
	}

	return nil
}

//Sign -TODO-
func (w *Wallets) Sign(id int64, ack *string) error {
	var err error

	var replyWallet mutils.WalletItem

	*ack = "ERROR"

	defer w.cleanUp()
	err = w.initStage1(id)
	if err != nil {
		return err
	}

	err = w.c.Call("Gateway.GetWalletByID", id, &replyWallet)
	if err != nil {
		return gutils.FormatErrorS("Gateway.GetWalletByID", "(%d) %v", id, err)
	}

	dataToSign := mutils.GetWalletSignature(&replyWallet)

	var args mutils.ArgsUpdateSignature

	args.Signature, err = gutils.GetSignature([]byte(dataToSign), keyAdmin)
	if err != nil {
		return gutils.FormatErrorS("GetSignature", "(%d) %v", id, err)
	}

	var reply string

	args.ID = id
	args.KeyID = certAdmin.SubjectKeyId
	args.Table = "users_userwallets"

	err = w.c.Call("Gateway.UpdateSignature", args, &reply)
	if err != nil {
		return gutils.FormatErrorS("Gateway.UpdateSignature", "(%d) %v", id, err)
	}

	*ack = "OK"

	return nil
}
