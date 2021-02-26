package main

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/seagiv/common/coinapi"
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/mpool/mutils"
)

var enabledPayServ bool

//PayServ --TODO--
type PayServ struct {
	ID int64

	сlientCN string

	c *gutils.ClientConnection

	withdrawal *mutils.WithdrawalItem

	cpOrig *mutils.UserCPItem
	cpNew  *mutils.UserCPItem

	locks []string
}

func (l *PayServ) onConnect(commonName string) bool {
	l.сlientCN = commonName

	if !enabledPayServ {
		return false
	}

	return gutils.IsIn(commonName, allowedCNs[connectPayServ])
}

func (l *PayServ) onDisconnect() {
}

func (l *PayServ) lock(name string) {
	txLocker.Lock(name)

	l.locks = append(l.locks, name)
}

func (l *PayServ) unlock() {
	for i := 0; i < len(l.locks); i++ {
		txLocker.Delete(l.locks[i])
	}
}

func (l *PayServ) checkWithdrawal(id int64) (*mutils.WithdrawalItem, error) {
	var err error

	//getting transaction data
	var replyWT mutils.WithdrawalItem
	err = l.c.Call("Gateway.GetWithdrawalTransactionByID", id, &replyWT)
	if err != nil {
		return nil, gutils.FormatErrorSI("Gateway.GetWithdrawalTransactionByID", l.ID, "%v", err)
	}

	gutils.RemoteLog.PutDebugI(id, "ID: %d, UserID: %d, CoinID: %d (%s), Status: %s, WalletID: %d, Amount: %s",
		replyWT.ID,
		replyWT.UserID,
		replyWT.CoinID, IDToTag(replyWT.CoinID),
		replyWT.Status,
		replyWT.WalletID,
		replyWT.Amount.String(),
	)

	//getting certificate from cache or from db and verify it
	cert, err := gutils.GetCertificate(l.c, replyWT.Signatures[replyWT.Status].KeyID, allowedCNs[replyWT.Status], replyWT.DateAdded)
	if err != nil {
		return nil, gutils.FatalError(err.Error())
	}

	//checking signature
	dataToCheck := mutils.GetWithdrawalSignature(&replyWT)

	err = gutils.VerifySignature([]byte(dataToCheck), replyWT.Signatures[replyWT.Status].Signature, cert.PublicKey.(*rsa.PublicKey))
	if err != nil {
		return nil, gutils.FormatFatalSI("VerifySignature", l.ID, "%v", err)
	}

	return &replyWT, nil
}

func (l *PayServ) getUserCP(userID, coinID int64) (*mutils.UserCPItem, error) {
	var err error

	args := mutils.ArgsUCIB{UserID: userID, CoinID: coinID}
	var replyGC mutils.UserCPItem
	err = l.c.Call("Gateway.GetUserCP", args, &replyGC)
	if err == nil {
		//getting certificate from cache or from db and verify it
		certP, err := gutils.GetCertificate(l.c, replyGC.KeyID, allowedCNs[tableUserCPs], replyGC.TimeStamp)
		if err != nil {
			return nil, gutils.FatalError(err.Error())
		}

		//checking signature
		dataToCheck := mutils.GetUserCPSignature(&replyGC)

		err = gutils.VerifySignature([]byte(dataToCheck), replyGC.Signature, certP.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return nil, gutils.FormatFatalSI("VerifySignature", l.ID, "%v", err)
		}

		if replyGC.LastBalanceID > 0 { // skipping this check for id = 0 income
			//checking userCP.LastBalanceID transaction have blockhash userCP.LastBlockHash
			var replyIT mutils.IncomeItem
			err = l.c.Call("Gateway.GetIncomeTransactionByID", replyGC.LastBalanceID, &replyIT)
			if err != nil {
				return nil, gutils.FormatErrorSI("Gateway.GetIncomeTransactionByID", l.ID, "%v", err)
			}

			if replyIT.BlockHash != replyGC.LastBlockHash {
				return nil, gutils.FormatFatalSI("Blockhash", l.ID, "userCP blockhash mismatch with userCP.LastBalanceID")
			}
		}
	} else {
		tempGC := storageUserCP.Get(userID, coinID)

		if (tempGC != nil) && (tempGC.ID != 0) {
			gutils.RemoteLog.PutInfoSI("GetUserCP", l.ID, "userCP for (%d, %d) not found, using locally cached", userID, coinID)

			replyGC = *tempGC
		} else {
			gutils.RemoteLog.PutInfoSI("GetUserCP", l.ID, "userCP for (%d, %d) not found, using empty", userID, coinID)

			replyGC.TimeStamp = time.Now()
			replyGC.CoinID = coinID
			replyGC.UserID = userID
		}
	}

	if !storageUserCP.Valid(replyGC) {
		return nil, gutils.FormatFatalSI("GlobalCheckStorage.Valid", l.ID, "userCP mismatch with cached ones")
	}

	return &replyGC, nil
}

func (l *PayServ) verifyUserCP(cp *mutils.UserCPItem) error {
	var err error

	var replyGCV mutils.ReplyUserCPVerification
	err = l.c.Call("Gateway.GetUserCPVerification", cp, &replyGCV)
	if err != nil {
		return gutils.FormatFatalSI("Gateway.GetUserCPVerification", l.ID, "%v", err)
	}

	if !cp.CreditAll.Equal(replyGCV.CreditAll) {
		return gutils.FormatFatalSI("creditAll", l.ID, "%s != %s", cp.CreditAll.String(), replyGCV.CreditAll.String())
	}

	if !cp.CreditUser.Equal(replyGCV.CreditUser) {
		return gutils.FormatFatalSI("creditUser", l.ID, "%s != %s", cp.CreditUser.String(), replyGCV.CreditUser.String())
	}

	if len(cp.KeyID) != 0 { // bypass this check on new userCP
		if !cp.SentAll.Equal(replyGCV.SentAll) {
			return gutils.FormatFatalSI("Sent", l.ID, "%s != %s", cp.SentAll.String(), replyGCV.SentAll.String())
		}
	} else {
		cp.SentAll = replyGCV.SentAll // fix for invalid SentAll on new userCPs created	during .Check
	}

	if !cp.Debit.Equal(replyGCV.Debit) {
		return gutils.FormatFatalSI("Debit", l.ID, "%s != %s", cp.Debit.String(), replyGCV.Debit.String())
	}

	if cp.LastBalanceID > replyGCV.LastBalanceID {
		return gutils.FormatFatalSI("LastBalanceID", l.ID, "check error %d > %d", cp.LastBalanceID, replyGCV.LastBalanceID)
	}

	if cp.LastWithdrawalID > replyGCV.LastWithdrawalID {
		return gutils.FormatFatalSI("LastWithdrawalID", l.ID, "check error %d > %d", cp.LastWithdrawalID, replyGCV.LastWithdrawalID)
	}

	return nil
}

func (l *PayServ) checkCoin(id int64) (*mutils.CoinItem, error) {
	var err error
	var reply mutils.CoinItem

	err = l.c.Call("Gateway.GetCoinByID", id, &reply)
	if err != nil {
		return nil, gutils.FormatFatalSI("Gateway.GetCoinByID", l.ID, "(%d)%v", id, err)
	}

	gutils.RemoteLog.PutDebugI(l.ID, "ID: %d, Tag: %s, Name: %v", reply.ID, strings.ToUpper(reply.Tag), reply.Name)

	//getting certificate from cache or from db and verify it
	certA, err := gutils.GetCertificate(l.c, reply.KeyID, allowedCNs[tableCoins], reply.DateModified)
	if err != nil {
		return nil, gutils.FormatFatalSI("getCertificate", l.ID, "%v", err)
	}

	dataToCheck := mutils.GetCoinSignature(&reply)

	err = gutils.VerifySignature([]byte(dataToCheck), reply.Signature, certA.PublicKey.(*rsa.PublicKey))
	if err != nil {
		return nil, gutils.FormatFatalSI("VerifySignature", l.ID, "(%d) error %v", reply.ID, err)
	}

	return &reply, nil
}

func (l *PayServ) checkWallet(walletID, userID, coinID int64) (*mutils.WalletItem, error) {
	var err error
	var reply mutils.WalletItem

	err = l.c.Call("Gateway.GetWalletByID", walletID, &reply)
	if err != nil {
		return nil, gutils.FormatFatalSI("Gateway.GetWalletByID", l.ID, "(%d)%v", walletID, err)
	}

	gutils.RemoteLog.PutDebugI(l.ID, "ID: %d, Name: %s, Address: %s", reply.ID, reply.Name, reply.Wallet)

	if reply.UserID != userID {
		return nil, gutils.FormatFatalSI("checkUserID", l.ID, "transaction and wallet userIDs mismatch (%d != %d)", userID, reply.UserID)
	}

	if reply.CoinID != coinID {
		return nil, gutils.FormatFatalSI("checkCoinID", l.ID, "transaction and wallet coinIDs mismatch (%d != %d)", coinID, reply.CoinID)
	}

	//getting certificate from cache or from db and verify it
	certA, err := gutils.GetCertificate(l.c, reply.KeyID, allowedCNs[tableWallets], reply.DateModified)
	if err != nil {
		if len(reply.KeyID) == 0 {
			return nil, gutils.FormatFatalSI("getCertificate", l.ID, "wallet.ID %d, - NO SIGNATURE [%s]\n", walletID, reply.KeyID)
		}

		return nil, gutils.FormatFatalSI("getCertificate", l.ID, "wallet.ID %d, - SIGNATUREKEY ID [%s] NOT FOUND\n", walletID, reply.KeyID)
	}

	dataToCheck := mutils.GetWalletSignature(&reply)

	err = gutils.VerifySignature([]byte(dataToCheck), reply.Signature, certA.PublicKey.(*rsa.PublicKey))
	if err != nil {
		return nil, gutils.FormatFatalSI("VerifySignature", l.ID, "(%d) error %v", reply.ID, err)
	}

	return &reply, nil
}

func (l *PayServ) declineWithdrawal(w *mutils.WithdrawalItem, comment string) error {
	var err error

	var args mutils.ArgsUpdateWithdrawal

	w.Comment = comment
	w.Status = statusProcessed

	args.ID = w.ID
	args.Status = w.Status
	args.Comment = w.Comment
	args.KeyID = certPayServ.SubjectKeyId
	args.UserID = w.UserID
	args.CoinID = w.CoinID

	args.Confirmed = time.Now()

	dataToSign := mutils.GetWithdrawalSignature(w)

	args.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorSI("GetSignature(P)", l.ID, "%v", err)
	}

	var declinedID int64

	err = l.c.Call("Gateway.DeclineWithdrawal", args, &declinedID)
	if err != nil {
		return gutils.FormatErrorSI("Gateway.DeclineWithdrawal", l.ID, "(%d) error %v", w.ID, err)
	}

	//getting declined transaction data
	var replyWT mutils.WithdrawalItem
	err = l.c.Call("Gateway.GetWithdrawalTransactionByID", declinedID, &replyWT)
	if err != nil {
		return gutils.FormatErrorSI("Gateway.GetWithdrawalTransactionByID", l.ID, "%v", err)
	}

	dataToSign = mutils.GetWithdrawalSignature(&replyWT)

	args.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorSI("GetSignature(D)", l.ID, "%v", err)
	}

	var reply string

	args.ID = replyWT.ID
	args.Status = replyWT.Status
	args.Comment = replyWT.Comment
	args.Confirmed = time.Now()

	err = l.c.Call("Gateway.UpdateWithdrawal", args, &reply)
	if err != nil {
		return gutils.FormatErrorSI("Gateway.UpdateWithdrawal", l.ID, "(%d) error %v", w.ID, err)
	}

	return nil
}

func (l *PayServ) createUserCP(cp *mutils.UserCPItem, id int64) error {
	var err error

	cp.KeyID = certPayServ.SubjectKeyId
	cp.TimeStamp = time.Now()

	dataToSign := mutils.GetUserCPSignature(cp)

	cp.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorSI("GetSignature", id, "%v", err)
	}

	var reply string

	err = l.c.Call("Gateway.CreateUserCP", *cp, &reply)
	if err != nil {
		return gutils.FormatErrorSI("Gateway.CreateUserCP", id, "(%d, %d) error %v", cp.UserID, cp.CoinID, err)
	}

	gutils.RemoteLog.PutDebugI(id, "(%d, %d)", cp.UserID, cp.CoinID)

	storageUserCP.Set(*cp)

	return nil
}

func (l *PayServ) isManualApprovalRequired(w *mutils.WithdrawalItem) bool {
	return true
}

func (l *PayServ) updateWithdrawal(w *mutils.WithdrawalItem, status, comment, hash string) error {
	var err error

	w.Status = status
	w.Comment = comment
	w.Hash = hash

	dataToSign := mutils.GetWithdrawalSignature(w)

	var args mutils.ArgsUpdateWithdrawal

	args.Signature, err = gutils.GetSignature([]byte(dataToSign), keyPayServ)
	if err != nil {
		return gutils.FormatErrorSI("GetSignature(D)", l.ID, "%v", err)
	}

	var reply string

	args.ID = w.ID
	args.Status = status
	args.Comment = w.Comment
	args.Hash = hash
	args.Confirmed = time.Now()
	args.KeyID = certPayServ.SubjectKeyId
	args.TxFee = w.TxFee

	err = l.c.Call("Gateway.UpdateWithdrawal", args, &reply)
	if err != nil {
		return gutils.FormatErrorSI("Gateway.UpdateWithdrawal", l.ID, "(%d) error %v", w.ID, err)
	}

	return nil
}

func (l *PayServ) initStage1(funcName string) error {
	var err error

	if !gutils.IsIn(l.сlientCN, allowedCNs[funcName]) {
		return gutils.RemoteLog.PutErrorSI("commonName", l.ID, "access DENIED to function %s for client [%s]", funcName, l.сlientCN)
	}

	gutils.RemoteLog.PutInfoI(l.ID, "CALL [%s]", l.сlientCN)

	l.c, err = gutils.InitClientTLS(
		addressDB,
		gutils.InfoTLS.ImCA,
		gutils.InfoTLS.Pair,
		cntlsDBGW,
		true,
	)
	if err != nil {
		return gutils.RemoteLog.PutErrorSI("InitClient", l.ID, "(%s) error %v", addressDB, err)
	}

	return nil
}

func (l *PayServ) initFull(w *mutils.WithdrawalItem, funcName string, statuses []string) error {
	var err error

	l.lock(getLockNameTx(l.ID))

	err = l.initStage1(funcName)
	if err != nil {
		return err
	}

	//STEP 01 checking transaction itself
	l.withdrawal, err = l.checkWithdrawal(l.ID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	l.lock(getLockNameCoin(l.withdrawal.CoinID))

	//checking transaction status
	if !gutils.IsIn(l.withdrawal.Status, statuses) {
		return gutils.RemoteLog.PutErrorSI("checkStatus", l.ID, "invalid transaction status (`%s` not in %v) for function %s", l.withdrawal.Status, statuses, funcName)
	}

	return nil
}

func (l *PayServ) cleanUp() error {
	l.unlock()

	if l.c != nil {
		/*if l.c.Connection.BytesRead > (10 * 1024 * 1024 * 1024) {
			gutils.RemoteLog.PutInfoI(l.ID, "excessive BytesRead in this connection %s", humanize.Bytes(l.c.Connection.BytesRead))
		}*/

		l.c.Close()
	}

	return nil
}

func (l *PayServ) checkAddressTo(coinID int64, addressTo string) error {
	api, err := coinapi.GetCoinAPI(l.ID, IDToTag(coinID))
	if err != nil {
		return nil
	}

	return api.IsValidAddress(addressTo)
}

//Process --TODO--
func (l *PayServ) Process(id int64, ack *string) error {
	var err error

	//HEADER++

	l.ID = id

	*ack = statusError

	defer l.cleanUp()
	err = l.initFull(l.withdrawal, funcProcess, []string{statusProcessing})
	if err != nil {
		return err
	}

	//HEADER--

	//STEP 01.01 checking coin
	_, err = l.checkCoin(l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	//STEP 01.02.01 checking wallet
	wallet, err := l.checkWallet(l.withdrawal.WalletID, l.withdrawal.UserID, l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	//wallet.Wallet = wallet.Wallet + "11" //!!TEST!!

	//STEP 01.02.02 checking wallet
	err = l.checkAddressTo(l.withdrawal.CoinID, wallet.Wallet)
	if err != nil {
		err = l.declineWithdrawal(l.withdrawal, "invalid wallet address")

		if err != nil {
			gutils.RemoteLog.PutInfoSI("declineWithdrawal", l.ID, "failed to update db (decline)", err)

			return err
		}

		*ack = statusDeclined

		gutils.RemoteLog.PutErrorSI("checkAddressTo", l.ID, "invalid wallet address [%s] - %s", wallet.Wallet, *ack)

		return nil
	}

	//STEP 02 getting and checking userCP
	l.cpOrig, err = l.getUserCP(l.withdrawal.UserID, l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}
	gutils.RemoteLog.PutDebugSI("getUserCP", l.ID, "(%d, %d, %d).ID %d",
		l.withdrawal.UserID,
		l.withdrawal.CoinID,
		l.cpOrig.LastBalanceID,
		l.cpOrig.ID,
	)

	err = l.verifyUserCP(l.cpOrig)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	cp := *l.cpOrig

	l.cpNew = &cp

	//STEP 03 getting balance, debit credit etc
	currentSaldo, err := getBalance(l.c, l.withdrawal.UserID, l.withdrawal.CoinID, l.cpNew, l.cpOrig, l.ID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	//STEP 04 checking proxypool transactions
	err = checkProxyPool(l.c, l.withdrawal.CoinID, l.cpOrig.LastBalanceID, l.ID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	//STEP 05 checking amount toward balance
	if (*currentSaldo).LessThan(decimal.Zero) {
		err = l.declineWithdrawal(l.withdrawal, "not enough funds")

		if err != nil {
			gutils.RemoteLog.PutInfoSI("declineWithdrawal", l.ID, "failed to update db (decline)", err)

			return err
		}

		*ack = statusDeclined

		gutils.RemoteLog.PutErrorSI("checkFunds", l.ID, "not enough funds (%s) - %s", currentSaldo.String(), *ack)

		return nil
	}

	//STEP 06 updating approved or pending
	if l.isManualApprovalRequired(l.withdrawal) {
		err = l.updateWithdrawal(l.withdrawal, statusPending, "", "")

		if err != nil {
			gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update db (pending)", err)

			return err
		}

		*ack = statusPending

		gutils.RemoteLog.PutInfoSI("pending", l.ID, "manual approval required - %s", *ack)
	} else {
		err = l.updateWithdrawal(l.withdrawal, statusApproved, "", "")

		if err != nil {
			gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update db (approved)", err)

			return err
		}

		*ack = statusApproved
	}

	//making userCP
	err = l.createUserCP(l.cpNew, l.ID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	return nil
}

//Decline --TODO--
func (l *PayServ) Decline(id int64, ack *string) error {
	var err error

	l.ID = id

	*ack = statusError

	defer l.cleanUp()
	err = l.initFull(l.withdrawal, funcDecline, []string{statusSending, statusPending, statusSendError, statusApproved})
	if err != nil {
		return err
	}

	err = l.declineWithdrawal(l.withdrawal, "declined by manager")

	if err != nil {
		gutils.RemoteLog.PutErrorSI("declineWithdrawal", l.ID, "failed to update db (decline)", err)

		return err
	}

	*ack = statusDeclined

	gutils.RemoteLog.PutInfoSI("manualDecline", l.ID, "declined by manager")

	return nil
}

//SetStatus --TODO--
func (l *PayServ) SetStatus(args mutils.ArgsIS, ack *string) error {
	var err error

	l.ID = args.ID

	*ack = statusError

	if args.S != statusSent {
		return gutils.RemoteLog.PutErrorI(l.ID, "status must be 'sent' only")
	}

	defer l.cleanUp()
	err = l.initFull(l.withdrawal, funcSetStatus, []string{statusPending, statusApproved, statusSending, statusSendError})
	if err != nil {
		return err
	}

	//STEP 02 getting and checking userCP
	l.cpOrig, err = l.getUserCP(l.withdrawal.UserID, l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}
	gutils.RemoteLog.PutDebugSI("getUserCP", l.ID, "(%d, %d, %d).ID %d",
		l.withdrawal.UserID,
		l.withdrawal.CoinID,
		l.cpOrig.LastBalanceID,
		l.cpOrig.ID,
	)

	err = l.verifyUserCP(l.cpOrig)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	cp := *l.cpOrig

	l.cpNew = &cp

	err = l.updateWithdrawal(l.withdrawal, statusSent, "", "")

	if err != nil {
		return gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update db (sent)", err)
	}

	l.cpNew.SentAll = l.cpNew.SentAll.Add(l.withdrawal.Amount)

	*ack = statusSent

	gutils.RemoteLog.PutInfoSI("sent", l.ID, "check successful - %s", *ack)

	//making userCP
	err = l.createUserCP(l.cpNew, l.ID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	return nil
}

//Approve --TODO--
func (l *PayServ) Approve(id int64, ack *string) error {
	var err error

	l.ID = id

	*ack = statusError

	defer l.cleanUp()
	err = l.initFull(l.withdrawal, funcApprove, []string{statusPending})
	if err != nil {
		return err
	}

	err = l.updateWithdrawal(l.withdrawal, statusApproved, "", "")

	if err != nil {
		gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update db (approve)", err)

		return err
	}

	*ack = statusApproved

	return err
}

//Send -TODO-
func (l *PayServ) Send(id int64, ack *string) error {
	var err error

	l.ID = id

	*ack = statusError

	defer l.cleanUp()
	err = l.initFull(l.withdrawal, funcSend, []string{statusApproved, statusSendError, statusSendRetry})
	if err != nil {
		return err
	}

	if coinapi.Coins[IDToTag(l.withdrawal.CoinID)] == nil {
		return gutils.RemoteLog.PutErrorSI("checkCoin", l.ID, "coin %d not allowed", l.withdrawal.CoinID)
	}

	//STEP 01 checking coin
	coin, err := l.checkCoin(l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutFatalSI("checkCoin", l.ID, "%v", err)
	}

	wal, err := l.checkWallet(l.withdrawal.WalletID, l.withdrawal.UserID, l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutFatalSI("checkWallet", l.ID, "%v", err)
	}

	if wal.UserID != l.withdrawal.UserID {
		return gutils.RemoteLog.PutFatalSI("checkWallet.UserID", l.ID, "wallet UserID %d does not match transaction UserID %d", wal.UserID, l.withdrawal.UserID)
	}

	if wal.CoinID != l.withdrawal.CoinID {
		return gutils.RemoteLog.PutFatalSI("checkWallet.CoinID", l.ID, "wallet CoinID %d does not match transaction CoinID %d", wal.CoinID, l.withdrawal.CoinID)
	}

	amountWF := l.withdrawal.Amount.Sub(l.withdrawal.Fee)

	if amountWF.LessThanOrEqual(decimal.Zero) {
		return gutils.FormatErrorSI("checkAmount", l.ID, "(amount - fee) zero or negative [%s]", amountWF.String())
	}

	var isRetry bool
	var txHash *string

	api, err := coinapi.GetCoinAPI(l.ID, IDToTag(coin.ID))
	if err != nil {
		return gutils.RemoteLog.PutErrorSI("GetCoinAPI", l.ID, "%v", err)
	}

	txHash, isRetry, l.withdrawal.TxFee, err = api.Send(amountWF, wal.Wallet)

	if err != nil {
		gutils.RemoteLog.PutErrorSI(coin.Tag, l.ID, "%v", err)

		if isRetry {
			err = l.updateWithdrawal(l.withdrawal, statusSendRetry, "insufficient funds, will retry later", "")

			if err != nil {
				return gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update status (send_retry)", err)
			}

			*ack = statusSendRetry
		} else {
			err = l.updateWithdrawal(l.withdrawal, statusSendError, err.Error(), "")

			if err != nil {
				return gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update status (send_error)", err)
			}

			*ack = statusSendError
		}

		return nil
	}

	if txHash == nil {
		gutils.RemoteLog.PutFatalSI(coin.Tag, l.ID, "unexpected error txHash is nil!")

		return nil
	}

	err = l.updateWithdrawal(l.withdrawal, statusSending, "", *txHash)
	if err != nil {
		gutils.RemoteLog.PutFatalSI("updateWithdrawal", l.ID, "failed to update status (sent)", err)

		return err
	}

	*ack = statusSending

	gutils.RemoteLog.PutInfoI(l.ID, "send successful - %s", *ack)

	return nil
}

func (l *PayServ) processResults(wItems *map[int64]*mutils.WithdrawalItem, ack *mutils.SendResults, status, comment, txHash string, fatal bool) {
	var err error

	for _, v := range *wItems {
		err = l.updateWithdrawal(v, status, comment, txHash)

		if err != nil {
			errText := fmt.Sprintf("failed to update status (%s) %v", status, err)

			if fatal {
				gutils.RemoteLog.PutFatalI(v.ID, errText)
			} else {
				gutils.RemoteLog.PutErrorI(v.ID, errText)
			}

			*ack = append(*ack, mutils.SendResult{ID: v.ID, Result: statusError, Comment: errText})
		} else {
			*ack = append(*ack, mutils.SendResult{ID: v.ID, Result: status, Comment: comment})
		}
	}
}

func (l *PayServ) processDecline(w *mutils.WithdrawalItem, ack *mutils.SendResults, comment string) {
	var err error

	err = l.declineWithdrawal(w, comment)
	if err != nil {
		errText := fmt.Sprintf("failed to update status (decline) %v", err)

		gutils.RemoteLog.PutFatalI(w.ID, errText)

		*ack = append(*ack, mutils.SendResult{ID: w.ID, Result: statusError, Comment: errText})
	} else {
		*ack = append(*ack, mutils.SendResult{ID: w.ID, Result: statusDeclined, Comment: comment})
	}
}

//SendCoin -TODO-
func (l *PayServ) SendCoin(coinID int64, ack *mutils.SendResults) error {
	var err error

	if coinapi.Coins[IDToTag(coinID)] == nil {
		return gutils.RemoteLog.PutErrorS("checkCoin", "coin %d not allowed", coinID)
	}

	defer l.cleanUp()
	err = l.initStage1(funcSend)
	if err != nil {
		return err
	}

	l.lock(getLockNameCoin(coinID))

	coin, err := l.checkCoin(coinID)
	if err != nil {
		return gutils.RemoteLog.PutFatalSI("checkCoin", coinID, "%v", err)
	}

	api, err := coinapi.GetCoinAPI(l.ID, IDToTag(coinID))
	if err != nil {
		return gutils.RemoteLog.PutErrorSI("GetCoinAPI", l.ID, "%v", err)
	}

	var replyWT mutils.Withdrawals

	err = l.c.Call("Gateway.GetReadyToSendWithdrawalsByCoin", coinID, &replyWT)
	if err != nil {
		return gutils.FormatFatalS("Gateway.GetReadyToSendWithdrawalsByCoin", "%v", err)
	}
	gutils.RemoteLog.PutDebugS("GetReadyToSendWithdrawalsByCoin", "(%d).TransactionCount %d", coinID, len(replyWT))

	for i := 0; i < len(replyWT); {
		var aItems coinapi.Transfers

		wItems := make(map[int64]*mutils.WithdrawalItem)

		l.lock(getLockNameTx(replyWT[i].ID))

		balance, err := api.GetBalance(api.GetServiceAddress())
		if err != nil {
			gutils.RemoteLog.PutErrorS("api.GetBalance", "%v", err)
		}
		gutils.RemoteLog.PutDebugS(strings.ToUpper(coin.Tag), "Balance(%s) %s", api.GetServiceAddress(), balance.String())

		var amountTotal decimal.Decimal
		var amountToSend decimal.Decimal

		for i < len(replyWT) {
			l.ID = replyWT[i].ID

			wal, err := l.checkWallet(replyWT[i].WalletID, replyWT[i].UserID, replyWT[i].CoinID)
			if err != nil {
				return gutils.RemoteLog.PutFatalSI("checkWallet", replyWT[i].ID, "%v", err)
			}

			if wal.UserID != replyWT[i].UserID {
				return gutils.RemoteLog.PutFatalSI("checkWallet.UserID", replyWT[i].ID, "wallet UserID %d does not match transaction UserID %d", wal.UserID, replyWT[i].UserID)
			}

			if wal.CoinID != replyWT[i].CoinID {
				return gutils.RemoteLog.PutFatalSI("checkWallet.CoinID", replyWT[i].ID, "wallet CoinID %d does not match transaction CoinID %d", wal.CoinID, replyWT[i].CoinID)
			}

			err = api.IsValidAddress(wal.Wallet)
			if err != nil {
				gutils.RemoteLog.PutErrorI(replyWT[i].ID, "%v", err)

				l.processDecline(&replyWT[i], ack, "invalid wallet address")

				i++

				continue
			}

			replyWT[i].Address = wal.Wallet

			amountToSend = replyWT[i].Amount.Sub(replyWT[i].Fee)

			if amountToSend.LessThanOrEqual(decimal.Zero) {
				errText := fmt.Sprintf("(amount - fee) zero or negative [%s]", amountToSend.String())

				gutils.RemoteLog.PutErrorSI("checkAmount", replyWT[i].ID, "%s", errText)

				l.processDecline(&replyWT[i], ack, errText)

				i++

				continue
			}

			amountTotal = amountTotal.Add(amountToSend)

			if (len(wItems) > 0) && amountTotal.GreaterThanOrEqual(balance) {
				gutils.RemoteLog.PutDebugS(strings.ToUpper(coin.Tag), "not enough balance for whole block (%s <= %s), splitting payments ...", balance.String(), amountTotal.String())

				break
			}

			wItems[replyWT[i].ID] = &replyWT[i]

			aItems = append(aItems, coinapi.Transfer{ID: replyWT[i].ID, Amount: amountToSend, Address: replyWT[i].Address})

			i++

			if int64(len(wItems)) >= coinapi.Coins[IDToTag(coinID)].OutLimit {
				break
			}
		}

		if len(wItems) > 0 {
			var isRetry bool
			var txHash *string

			txHash, isRetry, err = api.SendMany(&aItems)

			for _, v := range aItems {
				wItems[v.ID].TxFee = v.TxFee
			}

			if err != nil {
				gutils.RemoteLog.PutErrorS(coin.Tag, "%s", err)

				if isRetry {
					l.processResults(&wItems, ack, statusSendRetry, "insufficient funds, will retry later", "", false)
				} else {
					l.processResults(&wItems, ack, statusSendError, err.Error(), "", false)
				}
			} else {
				if txHash == nil {
					return gutils.RemoteLog.PutFatalS(coin.Tag, "unexpected error txHash is nil!")
				}

				l.processResults(&wItems, ack, statusSending, "", *txHash, true)

				gutils.RemoteLog.PutInfoS(coin.Tag, "send successful - %s", statusSending)
			}
		}
	}

	return nil
}

//Check -TODO-
func (l *PayServ) Check(id int64, ack *string) error {
	var err error
	var cr bool

	l.ID = id

	*ack = statusError

	defer l.cleanUp()
	err = l.initFull(l.withdrawal, funcCheck, []string{statusSending})
	if err != nil {
		return err
	}

	//STEP 02 getting and checking userCP
	l.cpOrig, err = l.getUserCP(l.withdrawal.UserID, l.withdrawal.CoinID)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}
	gutils.RemoteLog.PutDebugSI("getUserCP", l.ID, "(%d, %d, %d).ID %d",
		l.withdrawal.UserID,
		l.withdrawal.CoinID,
		l.cpOrig.LastBalanceID,
		l.cpOrig.ID,
	)

	err = l.verifyUserCP(l.cpOrig)
	if err != nil {
		return gutils.RemoteLog.PutError(err)
	}

	*ack = statusSending

	api, err := coinapi.GetCoinAPI(l.ID, IDToTag(l.withdrawal.CoinID))
	if err != nil {
		return gutils.RemoteLog.PutErrorSI("GetCoinAPI", l.ID, "%v", err)
	}

	gutils.RemoteLog.PutDebugI(l.ID, "txID %s", l.withdrawal.Hash)

	cr, l.withdrawal.TxFee, err = api.Check(l.withdrawal.Hash, l.withdrawal.TxFee)

	if err != nil {
		gutils.RemoteLog.PutInfoSI(IDToTag(l.withdrawal.CoinID), l.ID, "txid %s", l.withdrawal.Hash)
		gutils.RemoteLog.PutInfoSI(IDToTag(l.withdrawal.CoinID), l.ID, "failed: %v", err)
		gutils.RemoteLog.PutInfoSI(IDToTag(l.withdrawal.CoinID), l.ID, "check result - %s", *ack)

		return err
	}

	if !cr {
		err = l.updateWithdrawal(l.withdrawal, statusSendError, "transaction execution failed, manual processing required", l.withdrawal.Hash)

		if err != nil {
			gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update db (approve)", err)

			return err
		}

		*ack = statusSendError
	} else {
		//updating withdrawal status
		err = l.updateWithdrawal(l.withdrawal, statusSent, l.withdrawal.Comment, l.withdrawal.Hash)

		if err != nil {
			gutils.RemoteLog.PutErrorSI("updateWithdrawal", l.ID, "failed to update db (approve)", err)

			return err
		}

		//updating userCP
		cp := *l.cpOrig

		l.cpNew = &cp

		l.cpNew.SentAll = l.cpNew.SentAll.Add(l.withdrawal.Amount)

		//making userCP
		err = l.createUserCP(l.cpNew, l.ID)
		if err != nil {
			return gutils.RemoteLog.PutErrorSI("createUserCP", l.ID, "%v", err)
		}

		*ack = statusSent
	}

	gutils.RemoteLog.PutInfoSI("check", l.ID, "check result - %s", *ack)

	return err
}
