package main

import (
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/mpool/mutils"
)

//Coins --TODO--
type Coins struct {
	сlientCN string

	c *gutils.ClientConnection
}

func (c *Coins) onConnect(commonName string) bool {
	c.сlientCN = commonName

	return gutils.IsIn(commonName, allowedCNs[connectCoins])
}

func (c *Coins) onDisconnect() {
}

func (c *Coins) cleanUp() error {
	if c.c != nil {
		c.c.Close()
	}

	return nil
}

func (c *Coins) initStage1(id int64) error {
	var err error

	gutils.RemoteLog.PutInfoN("CALL(%d) [%s]", id, c.сlientCN)

	c.c, err = gutils.InitClientTLS(
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
func (c *Coins) Sign(id int64, ack *string) error {
	var err error

	var replyCoin mutils.CoinItem

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1(id)
	if err != nil {
		return err
	}

	err = c.c.Call("Gateway.GetCoinByID", id, &replyCoin)
	if err != nil {
		return gutils.FormatErrorS("Gateway.GetCoinByID", "(%d) %v", id, err)
	}

	dataToSign := mutils.GetCoinSignature(&replyCoin)

	var args mutils.ArgsUpdateSignature

	args.Signature, err = gutils.GetSignature([]byte(dataToSign), keyAdmin)
	if err != nil {
		return gutils.FormatErrorS("GetSignature", "(%d) %v", id, err)
	}

	var reply string

	args.ID = id
	args.KeyID = certAdmin.SubjectKeyId
	args.Table = "coins_coin"

	err = c.c.Call("Gateway.UpdateSignature", args, &reply)
	if err != nil {
		return gutils.FormatErrorS("Gateway.UpdateSignature", "(%d) %v", id, err)
	}

	*ack = "OK"

	return nil
}
