package main

import (
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/seagiv/common/coinapi"
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/foreign/jsonrpcf"
	"github.com/seagiv/mpool/mutils"

	_ "github.com/go-sql-driver/mysql"
)

const configPath = "/etc/mpool/pscli.conf"

var (
	//Options
	flagUserID = flag.Int64("userid", 0, "(param) user id")
	flagCoinID = flag.Int64("coinid", 0, "(param) coin id")
	flagPoolID = flag.Int64("poolid", 0, "(param) pool id")

	flagTxID = flag.String("txid", "", "(param) txid (hash)")

	flagAmount = flag.String("amount", "", "(param) amount")

	flagCert = flag.String("cert", "", "(param) certificate filename")
	flagKey  = flag.String("key", "", "(param) key filename")

	flagTransID = flag.Int("transid", 0, "(param) transaction id")

	//Debug
	flagProcessing = flag.Int64("process", 0, "(debug) process withdrawal transaction")
	flagApprove    = flag.Int64("approve", 0, "(debug) approve withdrawal transaction")
	flagDecline    = flag.Int64("decline", 0, "(debug) decline withdrawal transaction")
	flagCheck      = flag.Int64("check", 0, "(debug) check withdrawal transaction")
	flagSend       = flag.Int64("send", 0, "(debug) send withdrawal transaction")
	flagSetStatus  = flag.Int64("setstatus", 0, "(debug) set 'sent' status on a transaction")

	flagSendCoin = flag.Int64("sendcoin", 0, "(debug) send all withdrawal transactions for coin")

	flagSignCoin   = flag.Int64("signcoin", 0, "(debug) sign coin")
	flagSignWallet = flag.Int64("signwallet", 0, "(debug) sign wallet")

	flagDebugSendTo = flag.Bool("sendto", false, "(debug) sending test amount coins to address")

	//Utils
	flagEnable       = flag.Bool("enable", false, "(utils) enable PayServ")
	flagGetCoinAPI   = flag.Int64("getcoinapi", 0, "(utils) get coin api type")
	flagListLocks    = flag.Bool("listlocks", false, "(utils) list all locks")
	flagIsProcessing = flag.Int64("isprocessing", 0, "(utils) check if certain transaction process in progress")

	//Checks
	flagCheckProxyPool = flag.Int64("checkproxypool", 0, "(check) check proxypool (coin)")
	flagCheckCoins     = flag.Bool("checkcoins", false, "(check) check all coins")
	flagCheckWallets   = flag.Bool("checkwallets", false, "(check) check all wallets")
	flagCheckBalance   = flag.Bool("checkbalance", false, "(check) check balance (user, coin)")
	flagCheckBalances  = flag.Bool("checkbalances", false, "(check) check all balances")

	//!! DANGEROUS !!!
	flagCreateCorrection = flag.Bool("createcorrection", false, "(dangerous) create corrective transaction")

	//Local DB access
	localLoadCertFlag = flag.String("loadcert", "", "(local) parse and load cerficate into database")

	localSignTransFlag   = flag.Bool("signtrans", false, "(local) sign test transaction")
	localSignWalletsFlag = flag.Bool("signwallets", false, "(local) sign all wallets")

	localSignProxyPoolCorrectionsFlag = flag.Bool("signproxypoolcorrections", false, "(local) sign proxypool corrective transactions")
	localSignBalanceCorrectionsFlag   = flag.Bool("signbalancecorrections", false, "(local) sign balance corrective transactions")

	localCheckProxyPoolsFlag    = flag.Bool("checkproxypools", false, "(local, check) check all proxypools")
	localCheckBalanceErrorsFlag = flag.Bool("checkbalanceerrors", false, "(local) check balance errors")

	localTransInfoFlag = flag.Int64("transinfo", 0, "(local) print transaction info")

	localTestFlag = flag.Bool("test", false, "(local) test")

	localGetETHKeyFlag = flag.String("getethkey", "", "(utils) get private key from ethereum json key file")
)

func doProcess(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("PayServ.Process", *flagProcessing, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doApprove(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("PayServ.Approve", *flagApprove, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doCheck(c *gutils.ClientConnection, id int64) {
	var reply string

	//args := mutils.ArgsIS{ID: coinID, S: txID}

	err := c.Call("PayServ.Check", id, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doSetStatus(c *gutils.ClientConnection) {
	var reply string

	args := mutils.ArgsIS{ID: *flagSetStatus, S: "sent"}

	err := c.Call("PayServ.SetStatus", args, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doSend(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("PayServ.Send", *flagSend, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doSendCoin(c *gutils.ClientConnection, coinID int64) {
	var reply mutils.SendResults

	err := c.Call("PayServ.SendCoin", coinID, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: Count %d\n", len(reply))

		for i := 0; i < len(reply); i++ {
			fmt.Printf("[%d]: %d - %s (%s)\n", i, reply[i].ID, reply[i].Result, reply[i].Comment)
		}
	}
}

func doDecline(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("PayServ.Decline", *flagDecline, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doListLocks(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Utils.ListLocks", 0, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doIsProcessing(c *gutils.ClientConnection, id int64) {
	var reply bool

	err := c.Call("Utils.IsProcessing", id, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: %t\n", reply)
	}
}

func doSignCoin(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Coins.Sign", *flagSignCoin, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doSignWallet(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Wallets.Sign", *flagSignWallet, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doEnable(c *gutils.ClientConnection) {
	var reply string

	enabled := true

	err := c.Call("Utils.Enabled", enabled, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doGetCoinAPI(c *gutils.ClientConnection, coinID int64) {
	var reply string

	err := c.Call("Utils.GetCoinAPI", coinID, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doCheckProxyPool(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Check.ProxyPool", *flagCheckProxyPool, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doCheckBalance(c *gutils.ClientConnection, userID, coinID int64) {
	var reply string

	var args mutils.ArgsUCIB

	args.UserID = userID
	args.CoinID = coinID

	err := c.Call("Check.Balance", args, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doCheckBalances(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Check.Balances", 0, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doCheckCoins(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Check.Coins", 0, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func doCheckWallets(c *gutils.ClientConnection) {
	var reply string

	err := c.Call("Check.Wallets", 0, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

func loadCert(fileName string, db *sql.DB) {
	cert, err := gutils.LoadCertificateFromFile(fileName)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("SubjectKeyId: %X\n", cert.SubjectKeyId)
	fmt.Printf("Subject.CommonName: %s\n", cert.Subject.CommonName)
	fmt.Printf("AuthorityKeyId: %X\n", cert.AuthorityKeyId)
	fmt.Printf("Issuer.CommonName: %s\n", cert.Issuer.CommonName)
	fmt.Printf("NotBefore: %v\n", cert.NotBefore)
	fmt.Printf("NotAfter: %v\n", cert.NotAfter)
	fmt.Printf("isCA: %v\n", cert.IsCA)

	stmtIns, err := db.Prepare("INSERT INTO crypto_certs (subj_key_id, auth_key_id, not_before, not_after, subj_cn, issuer_cn, isCA, body)" +
		"VALUES(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatalf("loadCert.Prepare error %s", err.Error())
	}
	defer stmtIns.Close()

	_, err = stmtIns.Exec(
		cert.SubjectKeyId,
		cert.AuthorityKeyId,
		cert.NotBefore,
		cert.NotAfter,
		cert.Subject.CommonName,
		cert.Issuer.CommonName,
		cert.IsCA,
		cert.Raw,
	)
	if err != nil {
		log.Fatalf("loadCert.Exec error %s", err.Error())
	}
}

func localSignTrans(certfn, keyfn string, id int, db *sql.DB) {
	fmt.Printf("loading Cert ... ")

	cert, err := gutils.LoadCertificateFromFile(certfn)
	if err != nil {
		log.Fatalf("signTrans.LoadCertificate error %s", err.Error())
	}

	fmt.Printf("done\n")

	fmt.Printf("loading PrivateKey ... ")

	privateKey, err := gutils.LoadPrivateKeyFromFile(keyfn)
	if err != nil {
		log.Fatalf("signTrans.LoadPrivateKey error %s", err.Error())
	}

	fmt.Printf("done\n")

	var trans mutils.WithdrawalItem

	fmt.Printf("query transaction ... ")

	err = db.QueryRow(
		"SELECT "+
			"id, "+
			"status, "+
			"blockchain_transaction_id, "+
			"user_id, "+
			"coin_id, "+
			"wallet_id, "+
			"reward_type, "+
			"amount, "+
			"debit, "+
			"fee, "+
			"date_added "+
			"FROM finances_withdrawal_transaction WHERE id = ?", id).Scan(
		&trans.ID,
		&trans.Status,
		&trans.Hash,
		&trans.UserID,
		&trans.CoinID,
		&trans.WalletID,
		&trans.RewardType,
		&trans.Amount,
		&trans.Debit,
		&trans.Fee,
		&trans.DateAdded,
	)
	if err != nil {
		fmt.Printf("signTrans.QueryRow error %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("done\n")

	fmt.Printf("DateAdded: %v\n", trans.DateAdded)
	fmt.Printf("UserID: %d\n", trans.UserID)
	fmt.Printf("CoinID: %d\n", trans.CoinID)
	fmt.Printf("WalletID: %d\n", trans.WalletID)
	fmt.Printf("Amount: %s\n", trans.Amount)
	fmt.Printf("Debit: %s\n", trans.Debit)

	fmt.Printf("signing ... ")

	dataToSign := mutils.GetWithdrawalSignature(&trans)

	fmt.Printf("dataToSign[%s] ", dataToSign)

	signature, err := gutils.GetSignature([]byte(dataToSign), privateKey)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("done\n")

	fmt.Printf("updating db ... ")

	//_, err = db.Exec("UPDATE finances_withdrawal_transaction SET admin_sign = ?, admin_key_id = ? WHERE id = ?", signature, cert.SubjectKeyId, id)
	_, err = db.Exec("REPLACE INTO finances_withdrawal_signatures(withdraw_id, status, key_id, signature) VALUES(?, ?, ?, ?)",
		id,
		trans.Status,
		cert.SubjectKeyId,
		signature,
	)
	if err != nil {
		log.Fatalf("signTrans.Exec error %s", err.Error())
	}

	fmt.Printf("done\n")
}

type idUC struct {
	UserID int
	CoinID int
}

func getBalanceDiff(userID, coinID int, db *sql.DB) decimal.Decimal {
	var err error

	var nd decimal.NullDecimal

	var balanceUB decimal.Decimal
	var balanceUT decimal.Decimal
	var balanceWT decimal.Decimal
	var balanceFee decimal.Decimal

	err = db.QueryRow("SELECT balance FROM finances_user_balances WHERE user_id = ? AND coin_id = ?", userID, coinID).Scan(&nd)
	if err != nil {
		fmt.Printf("QueryRow(finances_user_balances, reg): %v\n", err)
		os.Exit(1)
	}

	if nd.Valid {
		balanceUB = nd.Decimal
	} else {
		balanceUB = decimal.Zero
	}

	if userID == 1 {
		err = db.QueryRow("SELECT SUM(amount) FROM finances_user_transaction WHERE coin_id = ? AND transaction_type_id = 8 AND status = 3", coinID).Scan(&nd)
		if err != nil {
			fmt.Printf("QueryRow(finances_user_balances, fee): %v\n", err)
			os.Exit(1)
		}

		if nd.Valid {
			balanceFee = nd.Decimal
		} else {
			balanceFee = decimal.Zero
		}
	}

	err = db.QueryRow("SELECT SUM(amount) FROM finances_user_transaction WHERE user_id = ? AND coin_id = ? AND transaction_type_id IN (1,12,13) AND status = 3",
		userID,
		coinID,
	).Scan(&nd)
	if err != nil {
		fmt.Printf("QueryRow: %v\n", err)
		os.Exit(1)
	}

	if nd.Valid {
		balanceUT = nd.Decimal
	} else {
		balanceUT = decimal.Zero
	}

	err = db.QueryRow("SELECT SUM(amount) FROM finances_withdrawal_transaction WHERE user_id = ? AND coin_id = ?", userID, coinID).Scan(&nd)
	if err != nil {
		fmt.Printf("QueryRow: %v\n", err)
		os.Exit(1)
	}

	if nd.Valid {
		balanceWT = nd.Decimal
	} else {
		balanceWT = decimal.Zero
	}

	saldo := balanceUT.Sub(balanceWT)

	diff := balanceUB.Sub(saldo)

	diff = diff.Sub(balanceFee)

	return diff
}

func checkBalanceErrors(db *sql.DB) error {
	fmt.Printf("Checking for balance errors ... \n")

	var item idUC

	rows, err := db.Query("SELECT user_id, coin_id FROM finances_user_balances")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.UserID,
			&item.CoinID,
		)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		diff := getBalanceDiff(item.UserID, item.CoinID, db)

		if !diff.Equal(decimal.Zero) {
			fmt.Printf("(%d, %d) -> %s \n", item.UserID, item.CoinID, diff.String())
		}
	}

	return nil
}

func createCorrection(c *gutils.ClientConnection, userID, coinID, poolID int64, amountS string) {
	var err error

	var args mutils.ArgsCorrection

	args.UserID = userID
	args.CoinID = coinID
	args.PoolID = poolID
	args.Amount, err = decimal.NewFromString(amountS)
	if err != nil {
		err := gutils.FormatErrorS("amount", "invalid string value [%s] for decimal", amountS)
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	var replyString string
	err = c.Call("Corrections.Apply", args, &replyString)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", replyString)
	}
}

func signProxyPoolCorrections(db *sql.DB, fnCert, fnKey string) {
	fmt.Printf("loadingCertificate ... ")

	cert, err := gutils.LoadCertificateFromFile(fnCert)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("done\n")

	fmt.Printf("loading PrivateKey ... ")

	key, err := gutils.LoadPrivateKeyFromFile(fnKey)
	if err != nil {
		log.Fatalf("LoadPrivateKey error %s", err.Error())
	}

	fmt.Printf("done\n")

	fmt.Printf("signProxyPool_Corrections ... \n")

	var item mutils.ProxyPoolItemCombined

	rows, err := db.Query(
		"SELECT " +
			"id," +
			"pool_id," +
			"coin_id," +
			"transaction_type_id, " +
			"reward_type, " +
			"blockhash," +
			"amount," +
			"balance," +
			"is_confirmed," +
			"date_added " +
			"FROM finances_proxypool_transactions WHERE transaction_type_id = 13 AND is_calculated = 1")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.PoolID,
			&item.CoinID,
			&item.TypeID,
			&item.RewardTypeID,
			&item.BlockHash,
			&item.AmountPPT,
			&item.Balance,
			&item.Confirmed,
			&item.DateAdded,
		)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		dataToSign := mutils.GetProxyPoolSignature(&item)

		fmt.Printf("-> %d [%s]\n", item.ID, dataToSign)

		signature, err := gutils.GetSignature([]byte(dataToSign), key)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = db.Exec("UPDATE finances_proxypool_transactions SET signature = ?, key_id = ? WHERE ID = ?",
			signature,
			cert.SubjectKeyId,
			item.ID,
		)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func signBalanceCorrections(db *sql.DB, fnCert, fnKey string) {
	fmt.Printf("loadingCertificate ... ")

	cert, err := gutils.LoadCertificateFromFile(fnCert)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("done\n")

	fmt.Printf("loading PrivateKey ... ")

	key, err := gutils.LoadPrivateKeyFromFile(fnKey)
	if err != nil {
		log.Fatalf("LoadPrivateKey error %s", err.Error())
	}

	fmt.Printf("done\n")

	fmt.Printf("signBalance_Corrections ... \n")

	var item mutils.IncomeItem

	rows, err := db.Query(
		"SELECT " +
			"id, " +
			"status, " +
			"amount, " +
			"transaction_type_id, " +
			"date_added, " +
			"blockhash, " +
			"pool_id, " +
			"user_id, " +
			"coin_id, " +
			"balance " +
			"FROM finances_user_transaction WHERE transaction_type_id = 13 ORDER BY ID")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.Status,
			&item.Amount,
			&item.TypeID,
			&item.DateAdded,
			&item.BlockHash,
			&item.PoolID,
			&item.UserID,
			&item.CoinID,
			&item.Balance,
		)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		dataToSign := mutils.GetIncomeSignature(&item)

		fmt.Printf("-> %d [%s]\n", item.ID, dataToSign)

		signature, err := gutils.GetSignature([]byte(dataToSign), key)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = db.Exec("UPDATE finances_user_transaction SET signature = ?, key_id = ? WHERE ID = ?",
			signature,
			cert.SubjectKeyId,
			item.ID,
		)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func checkProxyPools(c *gutils.ClientConnection, db *sql.DB) error {
	var err error
	var id int64
	var tag string
	var replyString string

	rows, err := db.Query("SELECT id, tag FROM coins_coin ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&id,
			&tag,
		)
		if err != nil {
			return err
		}

		err = c.Call("Check.ProxyPool", id, &replyString)
		if err != nil {
			return err
		}

		fmt.Printf("[%d, %s] -> %s\n", id, tag, replyString)
	}

	return err
}

func getTransInfo(c *gutils.ClientConnection, db *sql.DB, txID int64) error {
	var err error
	var tx mutils.WithdrawalItem
	var coin mutils.CoinItem
	var wallet mutils.WalletItem
	var replyAPI string
	var signature mutils.SignatureItem

	err = db.QueryRow(
		"SELECT "+
			"id, "+
			"status, "+
			"blockchain_transaction_id, "+
			"user_id, "+
			"coin_id, "+
			"wallet_id, "+
			"reward_type, "+
			"amount, "+
			"debit, "+
			"fee, "+
			"date_added "+
			"FROM finances_withdrawal_transaction WHERE id = ?", txID).Scan(
		&tx.ID,
		&tx.Status,
		&tx.Hash,
		&tx.UserID,
		&tx.CoinID,
		&tx.WalletID,
		&tx.RewardType,
		&tx.Amount,
		&tx.Debit,
		&tx.Fee,
		&tx.DateAdded,
	)
	if err != nil {
		fmt.Printf("QueryRow(tx) error %s", err.Error())
		os.Exit(1)
	}

	err = db.QueryRow(
		"SELECT "+
			"id, "+
			"key_id, "+
			"signature "+
			"FROM finances_withdrawal_signatures WHERE withdraw_id = ? AND status = ?",
		tx.ID,
		tx.Status,
	).Scan(
		&signature.ID,
		&signature.KeyID,
		&signature.Signature,
	)
	if err != nil {
		fmt.Printf("QueryRow(coin) error %s", err.Error())
		os.Exit(1)
	}

	err = db.QueryRow(
		"SELECT "+
			"id, "+
			"tag, "+
			"name "+
			"FROM coins_coin WHERE id = ?", tx.CoinID).Scan(
		&coin.ID,
		&coin.Tag,
		&coin.Name,
	)
	if err != nil {
		fmt.Printf("QueryRow(coin) error %s", err.Error())
		os.Exit(1)
	}

	err = db.QueryRow(
		"SELECT "+
			"id, "+
			"wallet "+
			"FROM users_userwallets WHERE id = ?", tx.WalletID).Scan(
		&wallet.ID,
		&wallet.Wallet,
	)
	if err != nil {
		fmt.Printf("QueryRow(coin) error %s", err.Error())
		os.Exit(1)
	}

	err = c.Call("Utils.GetCoinAPI", tx.CoinID, &replyAPI)
	if err != nil {
		fmt.Printf("GetCoinAPI error %s", err.Error())
		return err
	}

	fmt.Printf("ID: %d\n", tx.ID)
	fmt.Printf("UserID: %d\n", tx.UserID)
	fmt.Printf("CoinID: %d (%s - %s)\n", tx.CoinID, strings.ToUpper(coin.Tag), replyAPI)
	fmt.Printf("WalletID: %d, %s\n", tx.WalletID, wallet.Wallet)
	fmt.Printf("Amount: %s\n", tx.Amount.String())
	fmt.Printf("Status: %s\n", tx.Status)
	fmt.Printf("Hash: %s\n", tx.Hash)
	fmt.Printf("Date: %s\n", tx.DateAdded.UTC().String())
	fmt.Printf("KeyID: %s\n", hex.EncodeToString(signature.KeyID))
	fmt.Printf("Signature: %s\n", hex.EncodeToString(signature.Signature))

	return nil
}

func localSignWallets(fnCert, fnKey string, db *sql.DB) {
	fmt.Printf("loadingCertificate ... ")

	cert, err := gutils.LoadCertificateFromFile(fnCert)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("done\n")

	fmt.Printf("loading PrivateKey ... ")

	key, err := gutils.LoadPrivateKeyFromFile(fnKey)
	if err != nil {
		log.Fatalf("LoadPrivateKey error %s", err.Error())
	}

	fmt.Printf("done\n")

	fmt.Printf("signWallets ... \n")

	var item mutils.WalletItem

	rows, err := db.Query(
		"SELECT " +
			"id," +
			"name," +
			"wallet," +
			"date_added," +
			"date_modified," +
			"date_name_update," +
			"user_id," +
			"coin_id," +
			"is_active," +
			"is_deleted," +
			"key_id," +
			"signature " +
			"FROM users_userwallets")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Wallet,
			&item.DateAdded,
			&item.DateModified,
			&item.DateName,
			&item.UserID,
			&item.CoinID,
			&item.IsActive,
			&item.IsDeleted,
			&item.Signature,
			&item.KeyID,
		)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		if !item.IsActive {
			continue
		}

		fmt.Printf("-> %d ... ", item.ID)

		dataToSign := mutils.GetWalletSignature(&item)

		fmt.Printf("[%s]\n", dataToSign)

		signature, err := gutils.GetSignature([]byte(dataToSign), key)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = db.Exec("UPDATE users_userwallets SET signature = ?, key_id = ? WHERE ID = ?",
			signature,
			cert.SubjectKeyId,
			item.ID,
		)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func debugSendTo(c *gutils.ClientConnection) {
	var reply string

	args := gutils.ArgsDST{
		CoinTag:     coinapi.CoinMONA,
		AddressFrom: "-",
		AddressTo:   "-",
		PrivateKey:  "-",
		Amount:      gutils.OmmitError(decimal.NewFromString("25.0")).(decimal.Decimal),
	}

	err := c.Call("Debug.SendTo", args, &reply)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("REPLY: [%s]\n", reply)
	}
}

type replyAddress struct {
	IsValid      bool   `json:"isvalid"`
	Address      string `json:"address"`
	ScriptPubKey string `json:"scriptPubKey"`
	IsMine       bool   `json:"ismine"`
	IsWatchOnly  bool   `json:"iswatchonly"`
	IsScript     bool   `json:"isscript"`
	PubKey       string `json:"pubkey"`
	IsCompressed bool   `json:"iscompressed"`
	Account      string `json:"account"`
}

func getETHKey(fileName string) error {
	var err error

	keyJSON, err := ioutil.ReadFile(fileName)
	if err != nil {
		return gutils.FormatErrorS("ReadFile", "%v", err)
	}

	key, err := keystore.DecryptKey([]byte(keyJSON), "")
	if err != nil {
		return gutils.FormatErrorS("DecryptKey", "%v", err)
	}

	gutils.RemoteLog.PutInfoN("addressFrom [%s]\n", key.Address.String())
	gutils.RemoteLog.PutInfoN("privateKey [%s]\n", hex.EncodeToString(crypto.FromECDSA(key.PrivateKey)))

	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	//select {}

	flag.Parse()

	conf := readConfig(configPath)

	gutils.RemoteLog = gutils.InitLog("pscli")

	tlsPair, err := gutils.LoadX509KeyPairFromFile(conf.Crypto.CertTLS, conf.Crypto.KeyTLS)
	if err != nil {
		log.Fatalf("LoadX509KeyPairFromStorage(TLS): %v", err)
	}

	certImCA, err := gutils.LoadCertificateFromFile(conf.Crypto.ImCA)
	if err != nil {
		log.Fatalf("LoadCertificateFromFile(imCA): %v", err)
	}

	c, err := gutils.InitClientTLS(
		conf.AddressSVR,
		certImCA,
		tlsPair,
		"payserv",
		false,
	)
	if err != nil {
		log.Fatal(err)
	}

	if *flagProcessing != 0 {
		doProcess(c)
	}

	if *flagApprove != 0 {
		doApprove(c)
	}

	if *flagDecline != 0 {
		doDecline(c)
	}

	if *flagSend != 0 {
		doSend(c)
	}

	if *flagListLocks {
		doListLocks(c)
	}

	if *flagIsProcessing != 0 {
		doIsProcessing(c, *flagIsProcessing)
	}

	if *flagSendCoin != 0 {
		doSendCoin(c, *flagSendCoin)
	}

	if *flagCheck != 0 {
		/*if len(*flagTxID) == 0 {
			fmt.Println("ERROR: -txid is mandatory for check")
			os.Exit(1)
		}*/

		doCheck(c, *flagCheck)
	}

	if *flagSetStatus != 0 {
		doSetStatus(c)
	}

	if *flagSignCoin != 0 {
		doSignCoin(c)
	}

	if *flagSignWallet != 0 {
		doSignWallet(c)
	}

	if *flagEnable {
		doEnable(c)
	}

	if *flagGetCoinAPI != 0 {
		doGetCoinAPI(c, *flagGetCoinAPI)
	}

	if *flagCheckProxyPool != 0 {
		doCheckProxyPool(c)
	}

	if *flagCheckCoins {
		doCheckCoins(c)
	}

	if *flagCheckWallets {
		doCheckWallets(c)
	}

	if *flagCheckBalance {
		if (*flagUserID == 0) || (*flagCoinID == 0) {
			fmt.Println("ERROR: -userid, -coinid, are mandatory for check balance")
			os.Exit(1)
		}

		doCheckBalance(c, *flagUserID, *flagCoinID)
	}

	if *flagCheckBalances {
		doCheckBalances(c)
	}

	if *flagCreateCorrection {
		if (*flagUserID == 0) || (*flagCoinID == 0) || (*flagPoolID == 0) {
			fmt.Println("ERROR: -userid, -coinid -poolid are mandatory for create correction")
			os.Exit(1)
		}

		if len(*flagAmount) == 0 {
			fmt.Println("ERROR: -amount is mandatory for create correction")
			os.Exit(1)
		}

		createCorrection(c, *flagUserID, *flagCoinID, *flagPoolID, *flagAmount)
	}

	//local sql commands
	localCert := conf.Local.CertPath + *flagCert
	localKey := conf.Local.KeyPath + *flagKey

	if len(*localLoadCertFlag) != 0 {
		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		loadCert(*localLoadCertFlag, db)
	}

	if *localSignTransFlag {
		if (len(*flagCert) == 0) || (len(*flagKey) == 0) || (*flagTransID == 0) {
			fmt.Println("ERROR: -cert, -key -transid are mandatory for sign transaction")
			os.Exit(1)
		}

		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		localSignTrans(localCert, localKey, *flagTransID, db)
	}

	if *localSignWalletsFlag {
		if (len(*flagCert) == 0) || (len(*flagKey) == 0) {
			fmt.Println("ERROR: -cert and -key are mandatory for sign wallets")
			os.Exit(1)
		}

		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		localSignWallets(localCert, localKey, db)
	}

	if *localCheckBalanceErrorsFlag {
		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		checkBalanceErrors(db)
	}

	if *localSignProxyPoolCorrectionsFlag {
		if (len(*flagCert) == 0) || (len(*flagKey) == 0) {
			fmt.Println("ERROR: -cert, -key are mandatory for create correction")
			os.Exit(1)
		}

		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		signProxyPoolCorrections(db, localCert, localKey)
	}

	if *localSignBalanceCorrectionsFlag {
		if (len(*flagCert) == 0) || (len(*flagKey) == 0) {
			fmt.Println("ERROR: -cert, -key are mandatory for create correction")
			os.Exit(1)
		}

		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		signBalanceCorrections(db, localCert, localKey)
	}

	if *localCheckProxyPoolsFlag {
		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		checkProxyPools(c, db)
	}

	if *localTransInfoFlag != 0 {
		db, err := sql.Open("mysql", conf.Local.DataSource)
		if err != nil {
			log.Fatalf("sql.Open error %s", err.Error())
		}
		defer db.Close()

		getTransInfo(c, db, *localTransInfoFlag)
	}

	if *flagDebugSendTo {
		debugSendTo(c)
	}

	if *localTestFlag {
		test()
	}

	if len(*localGetETHKeyFlag) != 0 {
		err = getETHKey(*localGetETHKeyFlag)
		if err != nil {
			fmt.Printf("ERROR: %s", err.Error())
		}
	}
}
