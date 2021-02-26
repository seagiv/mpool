package main

import (
	"time"

	"github.com/seagiv/common/gutils"
	"github.com/seagiv/mpool/mutils"
)

//Corrections --TODO--
type Corrections struct {
	сlientCN string

	c *gutils.ClientConnection
}

func (c *Corrections) onConnect(commonName string) bool {
	c.сlientCN = commonName

	return gutils.IsIn(commonName, allowedCNs[connectCorrection])
}

func (c *Corrections) onDisconnect() {
}

func (c *Corrections) cleanUp() error {
	if c.c != nil {
		c.c.Close()
	}

	return nil
}

func (c *Corrections) initStage1() error {
	var err error

	gutils.RemoteLog.PutInfoN("CALL [%s]", c.сlientCN)

	c.c, err = gutils.InitClientTLS(
		addressDB,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
		cntlsDBGW,
		false,
	)

	if err != nil {
		return gutils.FormatErrorS("InitClient", "(%s) error %v", addressDB, err)
	}

	return nil
}

const maxErrors = 5
const retryTimeout = 30

//Apply -TODO-
func (c *Corrections) Apply(args mutils.ArgsCorrection, ack *string) error {
	var err error

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1()
	if err != nil {
		return err
	}

	var replyString = "-"
	var lockCounter int

	for {
		argsUCIB := mutils.ArgsUCIB{UserID: args.UserID, CoinID: args.CoinID}
		err = c.c.Call("Corrections.SetLock", argsUCIB, &replyString)
		if err != nil {
			return gutils.FormatErrorS("Corrections.SetLock", "(%d, %d, %s) %v", args.UserID, args.CoinID, args.Amount, err)
		}

		if replyString == "+" {
			break
		} else {
			lockCounter++

			if lockCounter > maxErrors {
				return gutils.FormatErrorS("Corrections.SetLock", "(%d) cannot acquire lock for proxypool", args.CoinID)
			}

			gutils.RemoteLog.PutDebugS("Corrections.SetLock", "(%d, %d) lock attempt failed, waiting %d seconds for retry", args.CoinID, lockCounter, retryTimeout)

			time.Sleep(retryTimeout * time.Second)
		}
	}

	var replyCorrection mutils.ReplyCorrection
	err = c.c.Call("Corrections.Apply", args, &replyCorrection)
	if err != nil {
		return gutils.FormatErrorS("Corrections.Apply", "(%d, %d, %s) %v", args.UserID, args.CoinID, args.Amount, err)
	}

	var dataToSign string
	var argsUS mutils.ArgsUpdateSignature

	//Sign Proxy Pool record
	var replyPP mutils.ProxyPoolItemCombined
	err = c.c.Call("Corrections.GetPPByID", replyCorrection.PoolID, &replyPP)
	if err != nil {
		return gutils.FormatErrorS("Corrections.GetPPByID", "(%d, %d, %s) %v", args.UserID, args.CoinID, args.Amount, err)
	}

	dataToSign = mutils.GetProxyPoolSignature(&replyPP)

	argsUS.ID = replyPP.ID
	argsUS.KeyID = certPayServ.SubjectKeyId
	argsUS.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorS("GetSignature(PP)", "%v", err)
	}

	argsUS.Table = "finances_proxypool_transactions"
	err = c.c.Call("Corrections.UpdateSignature", argsUS, &replyString)
	if err != nil {
		return gutils.FormatErrorS("Corrections.UpdateSignature", "(%s, %d) %v", argsUS.Table, argsUS.ID, err)
	}

	//Sign User Transaction record
	var replyUT mutils.IncomeItem
	err = c.c.Call("Corrections.GetUTByID", replyCorrection.IncomeID, &replyUT)
	if err != nil {
		return gutils.FormatErrorS("Corrections.GetUTByID", "(%d, %d, %s) %v", args.UserID, args.CoinID, args.Amount, err)
	}

	dataToSign = mutils.GetIncomeSignature(&replyUT)

	argsUS.ID = replyUT.ID
	argsUS.KeyID = certPayServ.SubjectKeyId
	argsUS.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorS("GetSignature(PP)", "%v", err)
	}

	argsUS.Table = "finances_user_transaction"
	err = c.c.Call("Corrections.UpdateSignature", argsUS, &replyString)
	if err != nil {
		return gutils.FormatErrorS("Corrections.UpdateSignature", "(%s, %d) %v", argsUS.Table, argsUS.ID, err)
	}

	err = c.c.Call("Corrections.Commit", 0, &replyString)
	if err != nil {
		return gutils.FormatErrorS("Corrections.Commit", "%v", err)
	}

	*ack = "OK"

	return nil
}
