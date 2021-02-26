package main

import (
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/mpool/mutils"
)

//Check --TODO--
type Check struct {
	сlientCN string

	c *gutils.ClientConnection
}

func (c *Check) onConnect(commonName string) bool {
	c.сlientCN = commonName

	return gutils.IsIn(commonName, allowedCNs[connectCheck])
}

func (c *Check) onDisconnect() {
}

func (c *Check) cleanUp() error {
	if c.c != nil {
		/*if c.c.Connection.BytesRead > 1*1014*1024 {
			gutils.RemoteLog.PutInfoN("excessive BytesRead in this connection %s", humanize.Bytes(c.c.Connection.BytesRead))
		}*/

		c.c.Close()
	}

	return nil
}

func (c *Check) initStage1() error {
	var err error

	gutils.RemoteLog.PutInfoN("CALL [%s]", c.сlientCN)

	c.c, err = gutils.InitClientTLS(
		addressDB,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
		cntlsDBGW,
		true,
	)
	if err != nil {
		return gutils.FormatErrorS("InitClientTLS", "(%s) error %v", addressDB, err)
	}

	return nil
}

//Coins -TODO-
func (c *Check) Coins(id int, ack *string) error {
	var err error
	var reply mutils.Coins

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1()
	if err != nil {
		return err
	}

	err = c.c.Call("Gateway.GetCoins", 0, &reply)
	if err != nil {
		return gutils.FormatErrorS("Gateway.GetCoins", "%v", err)
	}
	gutils.RemoteLog.PutDebugS("checkCoins", "CoinsCount %d", len(reply))

	for i := 0; i < len(reply); i++ {
		//getting certificate from cache or from db and verify it
		certA, err := gutils.GetCertificate(c.c, reply[i].KeyID, allowedCNs[tableCoins], reply[i].DateModified)
		if err != nil {
			return gutils.FormatErrorS("getCertificate", "%v", err)
		}

		dataToCheck := mutils.GetCoinSignature(&reply[i])

		err = gutils.VerifySignature([]byte(dataToCheck), reply[i].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return gutils.FormatErrorS("VerifySignature", "(%d) error %v", reply[i].ID, err)
		}

	}

	*ack = "OK"

	return nil
}

//Wallets -TODO-
func (c *Check) Wallets(id int, ack *string) error {
	var err error
	var reply mutils.Wallets
	var errorStr string

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1()
	if err != nil {
		return err
	}

	err = c.c.Call("Gateway.GetWallets", 0, &reply)
	if err != nil {
		return gutils.FormatErrorS("Gateway.GetWallets", "%v", err)
	}
	gutils.RemoteLog.PutDebugS("checkWallets", "WalletsCount %d", len(reply))

	for i := 0; i < len(reply); i++ {
		if !reply[i].IsActive {
			continue
		}

		//getting certificate from cache or from db and verify it
		certA, err := gutils.GetCertificate(c.c, reply[i].KeyID, allowedCNs[tableWallets], reply[i].DateModified)
		if err != nil {
			if len(reply[i].KeyID) == 0 {
				errorStr = errorStr + gutils.FormatErrorS("getCertificate", "wallet.ID %d (%d, %d), - NO SIGNATURE\n", reply[i].ID, reply[i].UserID, reply[i].CoinID).Error()
			} else {
				errorStr = errorStr + gutils.FormatErrorS("getCertificate", "wallet.ID %d (%d, %d), - KEY ID [%s] NOT FOUND\n", reply[i].ID, reply[i].UserID, reply[i].CoinID, hex.EncodeToString(reply[i].KeyID)).Error()
			}

			continue
		}

		dataToCheck := mutils.GetWalletSignature(&reply[i])

		err = gutils.VerifySignature([]byte(dataToCheck), reply[i].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			errorStr = errorStr + gutils.FormatErrorS("VerifySignature", "wallet.ID %d (%d, %d), - VERIFY ERROR\n", reply[i].ID, reply[i].UserID, reply[i].CoinID).Error()
		}

	}

	if len(errorStr) > 0 {
		*ack = "ERRORS FOUND:\n" + errorStr

		return nil
	}

	*ack = "OK"

	return nil
}

//Balances -TODO-
func (c *Check) Balances(id int64, ack *string) error {
	var err error
	var reply mutils.Users
	var errorStr string

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1()
	if err != nil {
		return err
	}

	err = c.c.Call("Gateway.GetUsers", 0, &reply)
	if err != nil {
		return gutils.FormatErrorS("Gateway.GetUsers", "%v", err)
	}
	gutils.RemoteLog.PutDebugS("checkBalances", "UsersCount %d", len(reply))

	for i := 0; i < len(reply); i++ {
		var cpOrig mutils.UserCPItem

		cpOrig.TimeStamp = time.Now()
		cpOrig.UserID = reply[i].UserID
		cpOrig.CoinID = reply[i].CoinID

		var cpNew mutils.UserCPItem

		_, err := getBalance(c.c, reply[i].UserID, reply[i].CoinID, &cpNew, &cpOrig, 0)
		if err != nil {
			errorStr = errorStr + gutils.FormatErrorS("getBalance", "(UserID: %d, CoinID: %d), - CHECK ERROR [%v]\n", reply[i].UserID, reply[i].CoinID, err).Error()
		}
	}

	if len(errorStr) > 0 {
		*ack = "ERRORS FOUND:\n" + errorStr

		return nil
	}

	*ack = "OK"

	return nil
}

//ProxyPool -TODO-
func (c *Check) ProxyPool(coinID int64, ack *string) error {
	var err error

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1()
	if err != nil {
		return err
	}

	err = checkProxyPool(c.c, coinID, 0, 0)
	if err != nil {
		return err
	}

	*ack = "OK"

	return nil
}

func (c *Check) createUserCP(cp *mutils.UserCPItem) error {
	var err error

	cp.KeyID = certPayServ.SubjectKeyId
	cp.TimeStamp = time.Now()

	dataToSign := mutils.GetUserCPSignature(cp)

	cp.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorS("GetSignature", "%v", err)
	}

	var reply string

	err = c.c.Call("Gateway.CreateUserCP", *cp, &reply)
	if err != nil {
		return gutils.FormatErrorS("Gateway.CreateUserCP", "(%d, %d) error %v", cp.UserID, cp.CoinID, err)
	}

	gutils.RemoteLog.PutDebugS("userCP", "(%d,%d)", cp.UserID, cp.CoinID)

	storageUserCP.Set(*cp)

	return nil
}

//Balance -TODO-
func (c *Check) Balance(args mutils.ArgsUCIB, ack *string) error {
	var err error

	*ack = "ERROR"

	defer c.cleanUp()
	err = c.initStage1()
	if err != nil {
		return err
	}

	var cpOrig mutils.UserCPItem

	cpOrig.TimeStamp = time.Now()
	cpOrig.UserID = args.UserID
	cpOrig.CoinID = args.CoinID

	var cpNew mutils.UserCPItem

	balance, err := getBalance(c.c, args.UserID, args.CoinID, &cpNew, &cpOrig, 0)
	if err != nil {
		return err
	}

	*ack = fmt.Sprintf("OK %s", balance.String())

	return nil
}

func getBalance(c *gutils.ClientConnection, userID, coinID int64, cpNew, cpOrig *mutils.UserCPItem, txID int64) (*decimal.Decimal, error) {
	var err error

	currentID := cpOrig.LastBalanceID
	recordsCount := 1
	totalCount := 0

	var args mutils.ArgsUCIB

	var aCNs []string

	var replyIT mutils.Incomes

	var debugData string

	for recordsCount > 0 {
		//getting user balances
		args = mutils.ArgsUCIB{UserID: userID, CoinID: coinID, FromID: currentID, Limit: limitSerial}

		err = c.Call("Gateway.GetIncomeTransactions", args, &replyIT)
		if err != nil {
			return nil, gutils.FormatFatalSI("Gateway.GetIncomeTransactions", txID, "%v", err)
		}

		recordsCount = len(replyIT)

		for i := 0; i < recordsCount; i++ {
			if replyIT[i].TypeID == ttCorrection { // workaround for corrections
				aCNs = allowedCNs[tableIncomesC]
			} else {
				aCNs = allowedCNs[tableIncomesN]
			}

			debugData = fmt.Sprintf("it.ID: %d", replyIT[i].ID)

			//getting certificate from cache or from db and verify it
			certA, err := gutils.GetCertificate(c, replyIT[i].KeyID, aCNs, replyIT[i].DateAdded)
			if err != nil {
				return nil, gutils.FormatFatalF("getCertificate", txID, debugData, "(balance) error %v", err)
			}

			dataToCheck := mutils.GetIncomeSignature(&replyIT[i])

			err = gutils.VerifySignature([]byte(dataToCheck), replyIT[i].Signature, certA.PublicKey.(*rsa.PublicKey))
			if err != nil {
				return nil, gutils.FormatFatalF("VerifySignature", txID, debugData, "(balance) [%s] error %v", dataToCheck, err)
			}

			if !replyIT[i].Balance.Equal(cpNew.CreditAll) {
				return nil, gutils.FormatFatalF("checkBalance", txID, debugData, "balance check (%s != %s) error",
					replyIT[i].Balance.String(),
					cpNew.CreditAll.String(),
				)
			}

			cpNew.CreditAll = cpNew.CreditAll.Add(replyIT[i].Amount)

			if (replyIT[i].TypeID == ttCredit) || (replyIT[i].TypeID == ttCreditRef) || (replyIT[i].TypeID == ttCorrection) {
				cpNew.CreditUser = cpNew.CreditUser.Add(replyIT[i].Amount)
			}

			if replyIT[i].ID > cpNew.LastBalanceID {
				cpNew.LastBalanceID = replyIT[i].ID
				cpNew.LastBlockHash = replyIT[i].BlockHash
			}
		}

		if recordsCount > 0 {
			currentID = replyIT[recordsCount-1].ID

			totalCount = totalCount + recordsCount
		}

		if (totalCount > 0) && (totalCount%(10*limitSerial) == 0) {
			gutils.RemoteLog.PutDebugSI("Balance", txID, "(%d, %d, %d) very long request in progress, current position %d", userID, coinID, args.FromID, totalCount)
		}
	}

	debugData = ""

	gutils.RemoteLog.PutDebugSI("Balance", txID, "(%d, %d, %d).TransactionCount %d", userID, coinID, args.FromID, totalCount)
	gutils.RemoteLog.PutDebugSI("Balance", txID, "(%d, %d).Credit %s %s", userID, coinID, cpNew.CreditAll.String(), cpNew.CreditUser.String())

	var cpCoin *mutils.CoinCPItem

	if userID == 1 {
		cpCoin, err = getCoinCP(c, coinID, txID)
		if err != nil {
			return nil, err
		}

		currentID = cpCoin.LastIncomeID

		totalCount = 0
		recordsCount = 1

		var replyIT mutils.Incomes

		for recordsCount > 0 {
			args = mutils.ArgsUCIB{CoinID: coinID, FromID: currentID, Limit: limitSerial}

			err = c.Call("Gateway.GetIncomeFeeTransactionsByCoinID", args, &replyIT)
			if err != nil {
				return nil, gutils.FormatFatalSI("Gateway.GetIncomeFeeTransactionsByCoinID", txID, "%v", err)
			}

			recordsCount = len(replyIT)

			for i := 0; i < recordsCount; i++ {
				aCNs = allowedCNs[tableIncomesN]

				debugData = fmt.Sprintf("it.ID: %d", replyIT[i].ID)

				//getting certificate from cache or from db and verify it
				certA, err := gutils.GetCertificate(c, replyIT[i].KeyID, aCNs, replyIT[i].DateAdded)
				if err != nil {
					return nil, gutils.FormatFatalF("getCertificate", txID, debugData, "%v", err)
				}

				dataToCheck := mutils.GetIncomeSignature(&replyIT[i])

				err = gutils.VerifySignature([]byte(dataToCheck), replyIT[i].Signature, certA.PublicKey.(*rsa.PublicKey))
				if err != nil {
					return nil, gutils.FormatFatalF("VerifySignature", txID, debugData, "(income), %v", err)
				}

				cpCoin.LastIncomeFee = cpCoin.LastIncomeFee.Add(replyIT[i].Amount)
			}

			if recordsCount > 0 {
				currentID = replyIT[recordsCount-1].ID
				totalCount = totalCount + recordsCount

				cpCoin.LastIncomeID = replyIT[recordsCount-1].ID
				cpCoin.LastIncomeBlockHash = replyIT[recordsCount-1].BlockHash
			}

			if (totalCount > 0) && (totalCount%(10*limitSerial) == 0) {
				gutils.RemoteLog.PutDebugSI("Fee", txID, "(%d, %d, %d) very long request in progress, current position %d", userID, coinID, args.FromID, totalCount)
			}

			err = createCoinCP(c, cpCoin, txID)
			if err != nil {
				return nil, err
			}
		}

		gutils.RemoteLog.PutDebugSI("Fee", txID, "(%d, %d, %d).TransactionCount (Fee) %d", userID, coinID, args.FromID, totalCount)
		gutils.RemoteLog.PutDebugSI("Fee", txID, "(%d, %d).Credit (Fee) %s %s", userID, coinID, cpNew.CreditAll.String(), cpCoin.LastIncomeFee.String())
	} else {
		cpCoin = &mutils.CoinCPItem{}
	}

	//getting user withdrawals
	var replyWT mutils.Withdrawals

	debit := decimal.Zero
	sentAll := decimal.Zero

	args = mutils.ArgsUCIB{UserID: userID, CoinID: coinID, FromID: cpOrig.LastWithdrawalID, Limit: 0}

	err = c.Call("Gateway.GetWithdrawalTransactions", args, &replyWT)
	if err != nil {
		return nil, gutils.FormatFatalSI("Gateway.GetWithdrawalTransactions", txID, "%v", err)
	}
	gutils.RemoteLog.PutDebugI(txID, "(%d, %d).TransactionCount %d", userID, coinID, len(replyWT))

	for i := 0; i < len(replyWT); i++ {
		debugData = fmt.Sprintf("wt.ID: %d", replyWT[i].ID)

		if len(replyWT[i].Signatures[replyWT[i].Status].KeyID) == 0 {
			return nil, gutils.FormatFatalF("VerifySignature", txID, debugData, "(withdrawal) signature not found for status '%s'", replyWT[i].Status)
		}

		//getting certificate from cache or from db and verify it (admin sign ?)
		certA, err := gutils.GetCertificate(c, replyWT[i].Signatures[replyWT[i].Status].KeyID, allowedCNs[replyWT[i].Status], replyWT[i].DateAdded)
		if err != nil {
			return nil, gutils.FormatFatalF("getCertificate", txID, debugData, "(withdrawal) error %v", err)
		}

		dataToCheck := mutils.GetWithdrawalSignature(&replyWT[i])

		err = gutils.VerifySignature([]byte(dataToCheck), replyWT[i].Signatures[replyWT[i].Status].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return nil, gutils.FormatFatalF("VerifySignature", txID, debugData, "(withdrawal, %s) error %v", replyWT[i].Status, err)
		}

		if !replyWT[i].Debit.Equal(debit) {
			return nil, gutils.FormatFatalF("checkDebit", txID, debugData, "(withdrawal) debit mismatch %s != %s", replyWT[i].Debit.String(), debit.String())
		}

		debit = debit.Add(replyWT[i].Amount)

		if replyWT[i].ID == cpNew.LastWithdrawalID {
			if !cpNew.Debit.Equal(debit) {
				return nil, gutils.FormatFatalF("checkDebit", txID, debugData, "(withdrawal) debit mismatch with userCP %s != %s", cpNew.Debit.String(), debit.String())
			}
		}

		if replyWT[i].ID > cpNew.LastWithdrawalID {
			cpNew.LastWithdrawalID = replyWT[i].ID
		}

		if replyWT[i].Status == statusSent {
			sentAll = sentAll.Add(replyWT[i].Amount)
		}
	}
	gutils.RemoteLog.PutDebugI(txID, "(%d, %d).Debit %s", userID, coinID, debit.String())

	debugData = ""

	cpNew.Debit = debit
	cpNew.SentAll = sentAll

	saldo := cpNew.CreditUser.Sub(debit)

	saldo = saldo.Add(cpCoin.LastIncomeFee)

	gutils.RemoteLog.PutDebugI(txID, "(%d, %d).Balance %s", userID, coinID, saldo.String())

	return &saldo, nil
}

func checkProxyPool(c *gutils.ClientConnection, coinID, fromID, txID int64) error {
	var err error

	gutils.RemoteLog.PutDebugSI("check", txID, "(%d, %d)", coinID, fromID)

	//getting proxypool transactions
	args := mutils.ArgsUCIB{CoinID: coinID, FromID: fromID}

	var replyPPT mutils.ProxyPoolsCombined

	//Check combined (stage 1)
	err = c.Call("Gateway.GetProxyPoolTransactionsJUT", args, &replyPPT)
	if err != nil {
		return gutils.FormatFatalSI("Gateway.GetProxyPoolTransactionsJUT", txID, "%v", err)
	}
	gutils.RemoteLog.PutDebugSI("GetProxyPoolTransactions(Combined)", txID, "(%d, %d).TransactionCount %d", args.CoinID, args.FromID, len(replyPPT))

	var aCNs []string

	var debugData string

	for i := 0; i < len(replyPPT); i++ {
		if replyPPT[i].TypeID == ttCorrection { // workaround for corrections
			aCNs = allowedCNs[tableProxyPoolC]
		} else {
			aCNs = allowedCNs[tableProxyPoolN]
		}

		debugData = fmt.Sprintf("pp.ID: %d, coinID: %d", replyPPT[i].ID, coinID)

		//getting certificate from cache or from db and verify it
		certA, err := gutils.GetCertificate(c, replyPPT[i].KeyID, aCNs, replyPPT[i].DateAdded)
		if err != nil {
			return gutils.FormatFatalF("gutils.GetCertificate(Combined)", txID, debugData, "%v", err)
		}

		dataToCheck := mutils.GetProxyPoolSignature(&replyPPT[i])

		err = gutils.VerifySignature([]byte(dataToCheck), replyPPT[i].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return gutils.FormatFatalF("VerifySignature(stage 1)", txID, debugData, "%v", coinID, err)
		}

		//checking pool transaction amount split into user balances
		if !replyPPT[i].AmountPPT.Equal(replyPPT[i].AmountUBT) {
			return gutils.FormatFatalF("amount(Combined)", txID, debugData, "proxypool balance check failed %s != %s", replyPPT[i].AmountPPT, replyPPT[i].AmountUBT)
		}
	}

	debugData = ""

	var balance decimal.Decimal
	var cpCoin *mutils.CoinCPItem

	cpCoin, err = getCoinCP(c, coinID, txID)
	if err != nil {
		return gutils.FatalError(err.Error())
	}

	args.FromID = cpCoin.LastPPID
	balance = cpCoin.LastPPBalance

	//Check pure (stage 2)
	err = c.Call("Gateway.GetProxyPoolTransactionsPure", args, &replyPPT)
	if err != nil {
		return gutils.FormatFatalSI("Gateway.GetProxyPoolTransactionsPure", txID, "%v", err)
	}
	gutils.RemoteLog.PutDebugSI("GetProxyPoolTransactions(Pure)", txID, "(%d, %d).TransactionCount %d", args.CoinID, args.FromID, len(replyPPT))

	for i := 0; i < len(replyPPT); i++ {
		if replyPPT[i].TypeID == ttCorrection { // workaround for corrections
			aCNs = allowedCNs[tableProxyPoolC]
		} else {
			aCNs = allowedCNs[tableProxyPoolN]
		}

		debugData = fmt.Sprintf("pp.ID: %d, coinID: %d", replyPPT[i].ID, coinID)

		//getting certificate from cache or from db and verify it
		certA, err := gutils.GetCertificate(c, replyPPT[i].KeyID, aCNs, replyPPT[i].DateAdded)
		if err != nil {
			return gutils.FormatFatalF("gutils.GetCertificate(Pure)", txID, debugData, "%v", err)
		}

		dataToCheck := mutils.GetProxyPoolSignature(&replyPPT[i])

		err = gutils.VerifySignature([]byte(dataToCheck), replyPPT[i].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return gutils.FormatFatalF("VerifySignature(stage 2)", txID, debugData, "%v", err)
		}

		//checking pool transaction balances
		if !balance.Equal(replyPPT[i].Balance) {
			return gutils.FormatFatalF("amount(Pure)", txID, debugData, "proxypool balance check failed %s != %s", balance.String(), replyPPT[i].Balance)
		}

		balance = balance.Add(replyPPT[i].AmountPPT)

		cpCoin.LastPPID = replyPPT[i].ID
		cpCoin.LastPPBalance = balance
		cpCoin.LastPPBlockHash = replyPPT[i].BlockHash
	}

	debugData = ""

	err = createCoinCP(c, cpCoin, txID)
	if err != nil {
		return gutils.FatalError(err.Error())
	}

	//Check ghosts (stage 3)
	var ghosts mutils.Ghosts

	err = c.Call("Gateway.GetProxyPoolTransactionsGhost", args, &ghosts)
	if err != nil {
		return gutils.FormatFatalSI("Gateway.GetProxyPoolTransactionsGhost", txID, "%v", err)
	}
	if len(ghosts) > 0 {
		return gutils.FormatFatalSI("ghost", txID, "ghost blockhash found in user transaction(s) first is (%d, %s)", ghosts[0].ID, ghosts[0].BlockHash)
	}

	return nil
}

func getCoinCP(c *gutils.ClientConnection, coinID, txID int64) (*mutils.CoinCPItem, error) {
	var err error

	var replyGC mutils.CoinCPItem
	err = c.Call("Gateway.GetCoinCP", coinID, &replyGC)
	if err == nil {
		//getting certificate from cache or from db and verify it
		certP, err := gutils.GetCertificate(c, replyGC.KeyID, allowedCNs[tableCoinCPs], replyGC.TimeStamp)
		if err != nil {
			return nil, gutils.FatalError(err.Error())
		}

		//checking signature
		dataToCheck := mutils.GetCoinCPSignature(&replyGC)

		err = gutils.VerifySignature([]byte(dataToCheck), replyGC.Signature, certP.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return nil, gutils.FormatErrorSI("VerifySignature", txID, "%v", err)
		}

		if replyGC.LastPPID > 0 { // skipping this check for id = 0 income
			//checking replyGC.LastPPID transaction have blockhash coinCP.LastPPBlockHash
			var replyPP mutils.ProxyPoolItemCombined
			err = c.Call("Gateway.GetProxyPoolTransactionByID", replyGC.LastPPID, &replyPP)
			if err != nil {
				return nil, gutils.FormatErrorSI("Gateway.GetProxyPoolTransactionByID", txID, "%v", err)
			}

			if replyPP.BlockHash != replyGC.LastPPBlockHash {
				return nil, gutils.FormatErrorSI("Blockhash", txID, "ProxyPool blockhash mismatch with coinCP.LastPPBlockHash %v", replyPP)
			}
		}

		if replyGC.LastIncomeID > 0 { // skipping this check for id = 0 income
			//checking replyGC.LastIncomeID transaction have blockhash coinCP.LastIncomeBlockHash
			var replyIT mutils.IncomeItem
			err = c.Call("Gateway.GetIncomeTransactionByID", replyGC.LastIncomeID, &replyIT)
			if err != nil {
				return nil, gutils.FormatErrorSI("Gateway.GetIncomeTransactionByID", txID, "%v", err)
			}

			if replyIT.BlockHash != replyGC.LastIncomeBlockHash {
				return nil, gutils.FormatErrorSI("Blockhash", txID, "Income blockhash mismatch with coinCP.LastIncomeBlockHash %v", replyIT)
			}
		}
	} else {
		tempGC := storageCoinCP.Get(coinID)

		if (tempGC != nil) && (tempGC.ID != 0) {
			gutils.RemoteLog.PutWarningSI("GetCoinCP", txID, "(%d) not found, using locally cached", coinID)

			replyGC = *tempGC
		} else {
			gutils.RemoteLog.PutInfoSI("GetCoinCP", txID, "(%d) not found, using empty", coinID)

			replyGC.TimeStamp = time.Now()
			replyGC.CoinID = coinID
		}
	}

	if !storageCoinCP.Valid(replyGC) {
		return nil, gutils.FormatErrorSI("GlobalCheckStorage.Valid", txID, "mismatch with cached ones")
	}

	return &replyGC, nil
}

func createCoinCP(c *gutils.ClientConnection, cp *mutils.CoinCPItem, txID int64) error {
	var err error

	cp.KeyID = certPayServ.SubjectKeyId
	cp.TimeStamp = time.Now()

	dataToSign := mutils.GetCoinCPSignature(cp)

	cp.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorSI("GetSignature", txID, "%v", err)
	}

	var reply string

	err = c.Call("Gateway.CreateCoinCP", *cp, &reply)
	if err != nil {
		return gutils.FormatErrorSI("Gateway.CreateCoinCP", txID, "(%d),  %v", cp.CoinID, err)
	}

	gutils.RemoteLog.PutDebugI(txID, "(%d)", cp.CoinID)

	storageCoinCP.Set(*cp)

	return nil
}
