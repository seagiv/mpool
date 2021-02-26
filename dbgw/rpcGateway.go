package main

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/mpool/mutils"
)

//Gateway -TODO-
type Gateway struct {
	сlientCN string

	db *sql.DB
}

func (r *Gateway) onConnect(commonName string) bool {
	var err error

	r.сlientCN = commonName

	if !gutils.IsIn(commonName, allowedCNs[connectGateway]) {
		return false
	}

	if len(dataSource) == 0 {
		return false
	}

	r.db, err = sql.Open("mysql", dataSource)
	if err != nil {
		gutils.RemoteLog.PutErrorN("sql.Open error %s", err.Error())

		return false
	}

	return true
}

func (r *Gateway) onDisconnect() {
	//gutils.RemoteLog.PutInfo("Gateway.onDisconnect")

	if r.db != nil {
		r.db.Close()
	}
}

//PutLog -TODO-
func (r *Gateway) PutLog(msg string, reply *string) error {
	*reply = ""

	gutils.RemoteLog.PutInfoN("%s", msg)

	return nil
}

//GetCertificateByKeyID --TODO--
func (r *Gateway) GetCertificateByKeyID(keyid []byte, reply *[]byte) error {
	err := r.db.QueryRow(
		"SELECT "+
			"body "+
			"FROM crypto_certs WHERE subj_key_id = ?", keyid).Scan(reply)

	if err != nil {
		gutils.RemoteLog.PutErrorN("(%X), %v", keyid, err)
	}

	return err
}

//GetWithdrawalSignaturesByID -TODO-
func (r *Gateway) GetWithdrawalSignaturesByID(id int64, reply *map[string]mutils.SignatureItem) error {
	rows, err := r.db.Query("SELECT id, timestamp, status, key_id, signature FROM finances_withdrawal_signatures WHERE withdraw_id = ?", id)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", id, err)

		return err
	}
	defer rows.Close()

	var sign mutils.SignatureItem
	var status string

	for rows.Next() {
		err := rows.Scan(
			&sign.ID,
			&sign.TimeStamp,
			&status,
			&sign.KeyID,
			&sign.Signature,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d), %v", id, err)

			return err
		}

		(*reply)[status] = sign
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d), %v", id, err)

		return err
	}

	return nil
}

//GetWithdrawalTransactionByID --TODO--
func (r *Gateway) GetWithdrawalTransactionByID(id int64, reply *mutils.WithdrawalItem) error {
	err := r.db.QueryRow(
		"SELECT "+
			"id, "+
			"parent_id, "+
			"status, "+
			"blockchain_transaction_id, "+
			"date_added, "+
			"coin_id, "+
			"user_id, "+
			"wallet_id, "+
			"reward_type, "+
			"amount, "+
			"debit, "+
			"tx_fee, "+
			"fee "+
			"FROM finances_withdrawal_transaction WHERE id = ?", id).Scan(
		&reply.ID,
		&reply.ParentID,
		&reply.Status,
		&reply.Hash,
		&reply.DateAdded,
		&reply.CoinID,
		&reply.UserID,
		&reply.WalletID,
		&reply.RewardType,
		&reply.Amount,
		&reply.Debit,
		&reply.TxFee,
		&reply.Fee,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", id, err)

		return err
	}

	reply.Signatures = make(map[string]mutils.SignatureItem)

	r.GetWithdrawalSignaturesByID(reply.ID, &reply.Signatures)
	if err != nil {
		gutils.RemoteLog.PutErrorS("GetWithdrawalSignaturesByID", "%v", err)
	}

	return err
}

//GetWithdrawalTransactions --TODO--
func (r *Gateway) GetWithdrawalTransactions(args mutils.ArgsUCIB, reply *mutils.Withdrawals) error {
	var item mutils.WithdrawalItem

	var count int

	rows, err := r.db.Query(
		"SELECT "+
			"id, "+
			"parent_id, "+
			"status, "+
			"blockchain_transaction_id, "+
			"date_added, "+
			"date_confirmed, "+
			"coin_id, "+
			"user_id, "+
			"wallet_id, "+
			"reward_type, "+
			"amount, "+
			"debit, "+
			"tx_fee, "+
			"fee "+
			"FROM finances_withdrawal_transaction "+
			"WHERE user_id = ? AND coin_id = ? "+
			"ORDER BY ID",
		args.UserID,
		args.CoinID,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d, %d), %v", args.UserID, args.CoinID, err)

		return err
	}
	defer rows.Close()

	var sqlTime mysql.NullTime

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.ParentID,
			&item.Status,
			&item.Hash,
			&item.DateAdded,
			&sqlTime,
			&item.CoinID,
			&item.UserID,
			&item.WalletID,
			&item.RewardType,
			&item.Amount,
			&item.Debit,
			&item.TxFee,
			&item.Fee,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d, %d, %d) error %v", item.ID, args.UserID, args.CoinID, err)

			return err
		}

		if sqlTime.Valid {
			item.DateConfirmed = sqlTime.Time
		} else {
			item.DateConfirmed = time.Time{}
		}

		item.Signatures = make(map[string]mutils.SignatureItem)

		r.GetWithdrawalSignaturesByID(item.ID, &item.Signatures)
		if err != nil {
			gutils.RemoteLog.PutErrorS("GetWithdrawalSignaturesByID", "%v", err)
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d, %d), %v", args.UserID, args.CoinID, err)

		return err
	}

	return err
}

//GetReadyToSendWithdrawalsByCoin -TODO-
func (r *Gateway) GetReadyToSendWithdrawalsByCoin(coinID int64, reply *mutils.Withdrawals) error {
	var item mutils.WithdrawalItem

	var count int

	request := "SELECT " +
		"id, " +
		"parent_id, " +
		"status, " +
		"blockchain_transaction_id, " +
		"date_added, " +
		"date_confirmed, " +
		"coin_id, " +
		"user_id, " +
		"wallet_id, " +
		"reward_type, " +
		"amount, " +
		"debit, " +
		"tx_fee, " +
		"fee " +
		"FROM finances_withdrawal_transaction " +
		"WHERE status IN ('approved', 'send_retry', 'send_error') AND coin_id = ? " +
		"ORDER BY ID"

	rows, err := r.db.Query(
		request,
		coinID,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", coinID, err)

		return err
	}
	defer rows.Close()

	var sqlTime mysql.NullTime

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.ParentID,
			&item.Status,
			&item.Hash,
			&item.DateAdded,
			&sqlTime,
			&item.CoinID,
			&item.UserID,
			&item.WalletID,
			&item.RewardType,
			&item.Amount,
			&item.Debit,
			&item.TxFee,
			&item.Fee,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d, %d) error %v", item.ID, coinID, err)

			return err
		}

		if sqlTime.Valid {
			item.DateConfirmed = sqlTime.Time
		} else {
			item.DateConfirmed = time.Time{}
		}

		item.Signatures = make(map[string]mutils.SignatureItem)

		r.GetWithdrawalSignaturesByID(item.ID, &item.Signatures)
		if err != nil {
			gutils.RemoteLog.PutErrorS("GetWithdrawalSignaturesByID", "%v", err)
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d), %v", coinID, err)

		return err
	}

	return err
}

//GetIncomeTransactionByID --TODO--
func (r *Gateway) GetIncomeTransactionByID(id int64, reply *mutils.IncomeItem) error {
	err := r.db.QueryRow(
		"SELECT "+
			"id,"+
			"status, "+
			"amount, "+
			"transaction_type_id, "+
			"date_added, "+
			"blockhash, "+
			"pool_id, "+
			"user_id, "+
			"coin_id, "+
			"balance, "+
			"signature, "+
			"key_id "+
			"FROM finances_user_transaction "+
			"WHERE ID = ?", id).Scan(
		&reply.ID,
		&reply.Status,
		&reply.Amount,
		&reply.TypeID,
		&reply.DateAdded,
		&reply.BlockHash,
		&reply.PoolID,
		&reply.UserID,
		&reply.CoinID,
		&reply.Balance,
		&reply.Signature,
		&reply.KeyID,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorN("(%d), %v", id, err)
	}

	return err
}

//GetIncomeTransactions --TODO--
func (r *Gateway) GetIncomeTransactions(args mutils.ArgsUCIB, reply *mutils.Incomes) error {
	var item mutils.IncomeItem

	var count int

	var request = "SELECT " +
		"id, " +
		"status, " +
		"amount, " +
		"transaction_type_id, " +
		"date_added, " +
		"blockhash, " +
		"pool_id, " +
		"user_id, " +
		"coin_id, " +
		"balance, " +
		"signature, " +
		"key_id " +
		"FROM finances_user_transaction " +
		"WHERE user_id = ? AND coin_id = ? AND transaction_type_id IN (" + creditAllU + ") AND status = 3 AND ID > ? " +
		"ORDER BY ID"

	if args.Limit > 0 {
		request = request + " LIMIT ?"
	}

	rows, err := r.db.Query(
		request,
		args.UserID,
		args.CoinID,
		args.FromID,
		args.Limit,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d, %d), %v", args.UserID, args.CoinID, err)

		return err
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
			&item.Signature,
			&item.KeyID,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d, %d), %v", args.UserID, args.CoinID, err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d, %d), %v", args.UserID, args.CoinID, err)

		return err
	}

	return err
}

//GetIncomeFeeTransactionsByCoinID --TODO--
func (r *Gateway) GetIncomeFeeTransactionsByCoinID(args mutils.ArgsUCIB, reply *mutils.Incomes) error {
	var item mutils.IncomeItem

	var count int

	var request = "SELECT " +
		"id, " +
		"status, " +
		"amount, " +
		"transaction_type_id, " +
		"date_added, " +
		"blockhash, " +
		"pool_id, " +
		"user_id, " +
		"coin_id, " +
		"balance, " +
		"signature, " +
		"key_id " +
		"FROM finances_user_transaction " +
		"WHERE coin_id = ? AND transaction_type_id = 8 AND status = 3 AND ID > ? " +
		"ORDER BY ID"

	if args.Limit > 0 {
		request = request + " LIMIT ?"
	}

	rows, err := r.db.Query(
		request,
		args.CoinID,
		args.FromID,
		args.Limit,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", args.CoinID, err)

		return err
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
			&item.Signature,
			&item.KeyID,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d), %v", args.CoinID, err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorN("fetch", "(%d), %v", args.CoinID, err)

		return err
	}

	return err
}

//GetProxyPoolTransactionByID -TODO-
func (r *Gateway) GetProxyPoolTransactionByID(id int64, reply *mutils.ProxyPoolItemCombined) error {
	err := r.db.QueryRow(
		"SELECT "+
			"date_added, "+
			"id, "+
			"amount, "+
			"coin_id, "+
			"pool_id, "+
			"transaction_type_id, "+
			"blockhash, "+
			"transaction_date, "+
			"reward_type, "+
			"balance, "+
			"pool_balance, "+
			"is_confirmed, "+
			"key_id, "+
			"signature "+
			"FROM finances_proxypool_transactions WHERE id = ?",
		id,
	).Scan(
		&reply.DateAdded,
		&reply.ID,
		&reply.AmountPPT,
		&reply.CoinID,
		&reply.PoolID,
		&reply.TypeID,
		&reply.BlockHash,
		&reply.TransactionDate,
		&reply.RewardTypeID,
		&reply.Balance,
		&reply.PoolBalance,
		&reply.Confirmed,
		&reply.KeyID,
		&reply.Signature,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", id, err)
	}

	return err
}

//GetProxyPoolTransactionsPure -TODO-
func (r *Gateway) GetProxyPoolTransactionsPure(args mutils.ArgsUCIB, reply *mutils.ProxyPoolsCombined) error {
	var item mutils.ProxyPoolItemCombined

	var count int

	rows, err := r.db.Query(
		"SELECT "+
			"date_added, "+
			"id, "+
			"amount, "+
			"coin_id, "+
			"pool_id, "+
			"transaction_type_id, "+
			"blockhash, "+
			"transaction_date, "+
			"reward_type, "+
			"balance, "+
			"pool_balance, "+
			"is_confirmed, "+
			"key_id, "+
			"signature "+
			"FROM finances_proxypool_transactions WHERE coin_id = ? AND id > ? AND transaction_type_id in (1,13) AND is_calculated = 1 AND is_confirmed IN (1,3) ORDER BY id ASC",
		args.CoinID,
		args.FromID,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", args.CoinID, err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.DateAdded,
			&item.ID,
			&item.AmountPPT,
			&item.CoinID,
			&item.PoolID,
			&item.TypeID,
			&item.BlockHash,
			&item.TransactionDate,
			&item.RewardTypeID,
			&item.Balance,
			&item.PoolBalance,
			&item.Confirmed,
			&item.KeyID,
			&item.Signature,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d), %v", args.CoinID, err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d), %v", args.CoinID, err)

		return err
	}

	return err
}

//GetProxyPoolTransactionsJUT -TODO-
func (r *Gateway) GetProxyPoolTransactionsJUT(args mutils.ArgsUCIB, reply *mutils.ProxyPoolsCombined) error {
	var item mutils.ProxyPoolItemCombined

	var count int

	rows, err := r.db.Query(
		"SELECT "+
			"PPT.ID, "+
			"PPT.blockhash, "+
			"PPT.date_added, "+
			"PPT.coin_id, "+
			"PPT.pool_id, "+
			"PPT.transaction_type_id, "+
			"PPT.reward_type, "+
			"PPT.amount, "+
			"SUM(UT.amount), "+
			"PPT.balance, "+
			"PPT.signature, "+
			"PPT.key_id, "+
			"PPT.is_confirmed "+
			"FROM finances_proxypool_transactions PPT JOIN finances_user_transaction UT "+
			"WHERE UT.coin_id = ? AND UT.transaction_type_id IN ("+creditAllU+") AND UT.status = 3 "+
			"AND PPT.blockhash IN (SELECT DISTINCT(blockhash) FROM finances_user_transaction WHERE transaction_type_id IN ("+creditAllU+") AND status = 3 AND coin_id = ? AND ID > ?) "+
			"AND PPT.is_calculated = 1 AND PPT.is_confirmed IN (1,3) "+
			"AND PPT.transaction_type_id IN (1,13) "+
			"AND PPT.coin_id = UT.coin_id "+
			"AND PPT.pool_id = UT.pool_id "+
			"AND PPT.blockhash = UT.blockhash "+
			"GROUP BY UT.coin_id, UT.pool_id, UT.blockhash "+
			"ORDER BY ID",
		args.CoinID,
		args.CoinID,
		args.FromID,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d, %d), %v", args.UserID, args.CoinID, err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.BlockHash,
			&item.DateAdded,
			&item.CoinID,
			&item.PoolID,
			&item.TypeID,
			&item.RewardTypeID,
			&item.AmountPPT,
			&item.AmountUBT,
			&item.Balance,
			&item.Signature,
			&item.KeyID,
			&item.Confirmed,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d, %d), %v", args.CoinID, args.FromID, err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d, %d), %v", args.CoinID, args.FromID, err)

		return err
	}

	return err
}

//GetProxyPoolTransactionsGhost -TODO-
func (r *Gateway) GetProxyPoolTransactionsGhost(args mutils.ArgsUCIB, reply *mutils.Ghosts) error {
	var item mutils.GhostItem

	var count int

	rows, err := r.db.Query(
		"SELECT id, blockhash FROM finances_user_transaction WHERE coin_id = ? "+
			"AND blockhash NOT IN (SELECT DISTINCT(blockhash) FROM finances_proxypool_transactions WHERE blockhash IS NOT NULL)",
		args.CoinID,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", args.CoinID, err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.BlockHash,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "(%d), %v", args.CoinID, err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d), %v", args.CoinID, err)

		return err
	}

	return err
}

//GetUserCP --TODO--
func (r *Gateway) GetUserCP(args mutils.ArgsUCIB, reply *mutils.UserCPItem) error {
	err := r.db.QueryRow(
		"SELECT "+
			"id, "+
			"user_id, "+
			"coin_id, "+
			"last_id_balance, "+
			"last_id_withdrawal, "+
			"last_blockhash, "+
			"timestamp, "+
			"credit_all, "+
			"credit_user, "+
			"debit, "+
			"sent_all, "+
			"signature, "+
			"key_id "+
			"FROM finances_user_balance_checkpoints WHERE user_id = ? AND coin_id = ?",
		args.UserID,
		args.CoinID,
	).Scan(
		&reply.ID,
		&reply.UserID,
		&reply.CoinID,
		&reply.LastBalanceID,
		&reply.LastWithdrawalID,
		&reply.LastBlockHash,
		&reply.TimeStamp,
		&reply.CreditAll,
		&reply.CreditUser,
		&reply.Debit,
		&reply.SentAll,
		&reply.Signature,
		&reply.KeyID,
	)

	if err == sql.ErrNoRows {
		gutils.RemoteLog.PutErrorS("scan", "(%d, %d) not found", args.UserID, args.CoinID)
	} else {
		gutils.RemoteLog.PutErrorS("scan", "(%d, %d), %v", args.UserID, args.CoinID, err)
	}

	return err
}

//GetUserCPVerification --TODO--
func (r *Gateway) GetUserCPVerification(args mutils.UserCPItem, reply *mutils.ReplyUserCPVerification) error {
	var sqlDebit decimal.NullDecimal
	var sqlSentAll decimal.NullDecimal
	var sqlCreditAll decimal.NullDecimal
	var sqlCreditUser decimal.NullDecimal

	var sqlLastBalanceID sql.NullInt64
	var sqlLastWithdrawalID sql.NullInt64

	err := r.db.QueryRow(
		"SELECT "+
			"(SELECT SUM(amount) FROM finances_withdrawal_transaction WHERE user_id = ? AND coin_id = ? AND id <= ?) AS debit,"+
			"(SELECT SUM(amount) FROM finances_withdrawal_transaction WHERE user_id = ? AND coin_id = ? AND status = 'sent') AS sentAll,"+
			"(SELECT SUM(amount) FROM finances_user_transaction WHERE transaction_type_id IN ("+creditAllU+") AND status = 3 AND user_id = ? AND coin_id = ? AND ID <= ?) AS creditAll,"+
			"(SELECT SUM(amount) FROM finances_user_transaction WHERE transaction_type_id IN ("+creditUserU+") AND status = 3 AND user_id = ? AND coin_id = ? AND ID <= ?) AS creditUser,"+
			"(SELECT MAX(ID) FROM finances_user_transaction WHERE user_id = ? AND coin_id = ?) AS lastBalanceID,"+
			"(SELECT MAX(ID) FROM finances_withdrawal_transaction WHERE user_id = ? AND coin_id = ?) AS lastWithdrawalID",
		args.UserID,
		args.CoinID,
		args.LastWithdrawalID,

		args.UserID,
		args.CoinID,

		args.UserID,
		args.CoinID,
		args.LastBalanceID,

		args.UserID,
		args.CoinID,
		args.LastBalanceID,

		args.UserID,
		args.CoinID,

		args.UserID,
		args.CoinID,
	).Scan(
		&sqlDebit,
		&sqlSentAll,
		&sqlCreditAll,
		&sqlCreditUser,
		&sqlLastBalanceID,
		&sqlLastWithdrawalID,
	)

	if sqlDebit.Valid {
		reply.Debit = sqlDebit.Decimal
	} else {
		reply.Debit = decimal.Zero
	}

	if sqlSentAll.Valid {
		reply.SentAll = sqlSentAll.Decimal
	} else {
		reply.SentAll = decimal.Zero
	}

	if sqlCreditAll.Valid {
		reply.CreditAll = sqlCreditAll.Decimal
	} else {
		reply.CreditAll = decimal.Zero
	}

	if sqlCreditUser.Valid {
		reply.CreditUser = sqlCreditUser.Decimal
	} else {
		reply.CreditUser = decimal.Zero
	}

	if sqlLastBalanceID.Valid {
		reply.LastBalanceID = sqlLastBalanceID.Int64
	} else {
		reply.LastBalanceID = 0
	}

	if sqlLastWithdrawalID.Valid {
		reply.LastWithdrawalID = sqlLastWithdrawalID.Int64
	} else {
		reply.LastWithdrawalID = 0
	}

	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d,%d), %v", args.UserID, args.CoinID, err)
	}

	return err
}

//GetUserCPs --TODO--
func (r *Gateway) GetUserCPs(id int64, reply *mutils.UserCPs) error {
	var item mutils.UserCPItem

	var count int

	rows, err := r.db.Query(
		"SELECT " +
			"id, " +
			"user_id, " +
			"coin_id, " +
			"last_id_balance, " +
			"last_id_withdrawal, " +
			"last_blockhash, " +
			"timestamp, " +
			"credit_all, " +
			"credit_user, " +
			"debit, " +
			"sent_all, " +
			"signature, " +
			"key_id " +
			"FROM finances_user_balance_checkpoints GROUP BY user_id, coin_id ORDER BY timestamp DESC")
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "%v", err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CoinID,
			&item.LastBalanceID,
			&item.LastWithdrawalID,
			&item.LastBlockHash,
			&item.TimeStamp,
			&item.CreditAll,
			&item.CreditUser,
			&item.Debit,
			&item.SentAll,
			&item.Signature,
			&item.KeyID,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "%v", err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "%v", err)

		return err
	}

	return err
}

//GetCoinCP --TODO--
func (r *Gateway) GetCoinCP(coinID int64, reply *mutils.CoinCPItem) error {
	err := r.db.QueryRow(
		"SELECT "+
			"id, "+
			"coin_id, "+
			"last_id_proxypool, "+
			"last_balance_proxypool, "+
			"last_blockhash_proxypool, "+
			"last_id_income, "+
			"last_fee_income, "+
			"last_blockhash_income, "+
			"timestamp, "+
			"signature, "+
			"key_id "+
			"FROM finances_coin_checkpoints WHERE coin_id = ?",
		coinID,
	).Scan(
		&reply.ID,
		&reply.CoinID,
		&reply.LastPPID,
		&reply.LastPPBalance,
		&reply.LastPPBlockHash,
		&reply.LastIncomeID,
		&reply.LastIncomeFee,
		&reply.LastIncomeBlockHash,
		&reply.TimeStamp,
		&reply.Signature,
		&reply.KeyID,
	)

	if err == sql.ErrNoRows {
		gutils.RemoteLog.PutErrorS("query", "(%d) not found", coinID)
	}

	return err
}

//GetCoinCPs --TODO--
func (r *Gateway) GetCoinCPs(id int64, reply *mutils.CoinCPs) error {
	var item mutils.CoinCPItem

	var count int

	rows, err := r.db.Query(
		"SELECT " +
			"id, " +
			"coin_id, " +
			"last_id_proxypool, " +
			"last_balance_proxypool, " +
			"last_blockhash_proxypool, " +
			"last_id_income, " +
			"last_fee_income, " +
			"last_blockhash_income, " +
			"timestamp, " +
			"signature, " +
			"key_id " +
			"FROM finances_coin_checkpoints ORDER BY id DESC")
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "%v", err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.CoinID,
			&item.LastPPID,
			&item.LastPPBalance,
			&item.LastPPBlockHash,
			&item.LastIncomeID,
			&item.LastIncomeFee,
			&item.LastIncomeBlockHash,
			&item.TimeStamp,
			&item.Signature,
			&item.KeyID,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "%v", err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "%v", err)

		return err
	}

	return err
}

//DeclineWithdrawal --TODO--
func (r *Gateway) DeclineWithdrawal(args mutils.ArgsUpdateWithdrawal, id *int64) error {
	tx, err := r.db.Begin()

	if err != nil {
		gutils.RemoteLog.PutErrorS("begin", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	_, err = tx.Exec("UPDATE finances_withdrawal_transaction SET "+
		"status = ?, "+
		"comment = ?, "+
		"date_confirmed = ? "+
		"WHERE id = ?",
		args.Status,
		args.Comment,
		args.Confirmed,
		args.ID,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("update", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	result, err := tx.Exec("INSERT INTO finances_withdrawal_transaction "+
		"(parent_id, amount, status, comment, date_added, date_confirmed, coin_id, user_id, modified_by_user_id, wallet_id, reward_type, debit) "+
		"SELECT ?, -amount, 'declined', ?, date_added, date_confirmed, coin_id, user_id, modified_by_user_id, wallet_id, reward_type, "+
		"(SELECT debit+W1.amount FROM finances_withdrawal_transaction W1 WHERE user_id = ? AND coin_id = ? ORDER BY ID DESC LIMIT 1) "+
		"FROM finances_withdrawal_transaction WHERE ID = ?",
		args.ID,
		args.Comment,
		args.UserID,
		args.CoinID,
		args.ID,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("insert", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	*id, _ = result.LastInsertId()

	_, err = tx.Exec("REPLACE INTO finances_withdrawal_signatures(withdraw_id, status, key_id, signature) VALUES(?, ?, ?, ?)",
		args.ID,
		args.Status,
		args.KeyID,
		args.Signature,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("exec", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	tx.Commit()

	return err
}

//UpdateWithdrawal --TODO--
func (r *Gateway) UpdateWithdrawal(args mutils.ArgsUpdateWithdrawal, reply *string) error {
	var err error

	tx, err := r.db.Begin()

	if err != nil {
		gutils.RemoteLog.PutErrorS("begin", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	_, err = tx.Exec("UPDATE finances_withdrawal_transaction SET "+
		"status = ?, "+
		"comment = ?, "+
		"date_confirmed = ?, "+
		"tx_fee = ?, "+
		"blockchain_transaction_id = ? "+
		"WHERE id = ?",
		args.Status,
		args.Comment,
		args.Confirmed,
		args.TxFee,
		args.Hash,
		args.ID,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("update", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	_, err = tx.Exec("REPLACE INTO finances_withdrawal_signatures(withdraw_id, status, key_id, signature) VALUES(?, ?, ?, ?)",
		args.ID,
		args.Status,
		args.KeyID,
		args.Signature,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorN("replace", "(%d), %v", args.ID, err)

		tx.Rollback()

		return err
	}

	tx.Commit()

	return err
}

//CreateUserCP --TODO--
func (r *Gateway) CreateUserCP(args mutils.UserCPItem, reply *string) error {
	_, err := r.db.Exec("REPLACE INTO "+
		"finances_user_balance_checkpoints("+
		"last_id_balance, "+
		"last_id_withdrawal, "+
		"last_blockhash, "+
		"user_id, "+
		"coin_id, "+
		"timestamp, "+
		"credit_all, "+
		"credit_user, "+
		"debit, "+
		"sent_all, "+
		"signature, "+
		"key_id"+
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		args.LastBalanceID,
		args.LastWithdrawalID,
		args.LastBlockHash,
		args.UserID,
		args.CoinID,
		args.TimeStamp,
		args.CreditAll,
		args.CreditUser,
		args.Debit,
		args.SentAll,
		args.Signature,
		args.KeyID,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("exec", "(%d, %d, %v), %v", args.UserID, args.CoinID, args.TimeStamp, err)
	}

	return err
}

//CreateCoinCP --TODO--
func (r *Gateway) CreateCoinCP(args mutils.CoinCPItem, reply *string) error {
	_, err := r.db.Exec("REPLACE INTO finances_coin_checkpoints("+
		"last_id_proxypool, "+
		"last_balance_proxypool, "+
		"last_blockhash_proxypool, "+
		"last_id_income, "+
		"last_fee_income, "+
		"last_blockhash_income, "+
		"coin_id, "+
		"timestamp, "+
		"signature, "+
		"key_id"+
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		args.LastPPID,
		args.LastPPBalance,
		args.LastPPBlockHash,
		args.LastIncomeID,
		args.LastIncomeFee,
		args.LastIncomeBlockHash,
		args.CoinID,
		args.TimeStamp,
		args.Signature,
		args.KeyID,
	)

	if err != nil {
		gutils.RemoteLog.PutErrorS("exec", "(%d, %v), %v", args.CoinID, args.TimeStamp, err)
	}

	return err
}

//GetCoins -TODO-
func (r *Gateway) GetCoins(id int64, reply *mutils.Coins) error {
	var item mutils.CoinItem

	var count int

	var icon sql.NullString

	rows, err := r.db.Query(
		"SELECT " +
			"id," +
			"date_added," +
			"date_modified," +
			"tag," +
			"name," +
			"is_active," +
			"algorithm_id," +
			"port," +
			"reward_type," +
			"fee," +
			"confirmations," +
			"description," +
			"icon," +
			"is_viewable," +
			"orphan_reward," +
			"withdrawal_fee," +
			"min_sum_withdrawal," +
			"key_id," +
			"signature " +
			"FROM coins_coin")
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "%v", err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.DateAdded,
			&item.DateModified,
			&item.Tag,
			&item.Name,
			&item.IsActive,
			&item.AlgorithmID,
			&item.Port,
			&item.RewardType,
			&item.FeeOur,
			&item.Confirmations,
			&item.Description,
			&icon,
			&item.IsViewable,
			&item.OrphanReward,
			&item.FeeWithdrawal,
			&item.MinSumWithdrawal,
			&item.KeyID,
			&item.Signature,
		)

		if icon.Valid {
			item.Icon = icon.String
		} else {
			item.Icon = ""
		}

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "%v", err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("scan", "%v", err)

		return err
	}

	return err
}

//GetCoinByID --TODO--
func (r *Gateway) GetCoinByID(id int64, reply *mutils.CoinItem) error {
	var icon sql.NullString

	err := r.db.QueryRow(
		"SELECT "+
			"id,"+
			"date_added,"+
			"date_modified,"+
			"tag,"+
			"name,"+
			"is_active,"+
			"algorithm_id,"+
			"port,"+
			"reward_type,"+
			"fee,"+
			"confirmations,"+
			"description,"+
			"icon,"+
			"is_viewable,"+
			"orphan_reward,"+
			"withdrawal_fee,"+
			"min_sum_withdrawal,"+
			"key_id,"+
			"signature "+
			"FROM coins_coin WHERE id = ?", id).Scan(
		&reply.ID,
		&reply.DateAdded,
		&reply.DateModified,
		&reply.Tag,
		&reply.Name,
		&reply.IsActive,
		&reply.AlgorithmID,
		&reply.Port,
		&reply.RewardType,
		&reply.FeeOur,
		&reply.Confirmations,
		&reply.Description,
		&icon,
		&reply.IsViewable,
		&reply.OrphanReward,
		&reply.FeeWithdrawal,
		&reply.MinSumWithdrawal,
		&reply.KeyID,
		&reply.Signature,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", id, err)

		return err
	}

	if icon.Valid {
		reply.Icon = icon.String
	} else {
		reply.Icon = ""
	}

	if err != nil {
		gutils.RemoteLog.PutErrorS("scan", "(%d), %v", id, err)

		return err
	}

	return nil
}

//GetWallets -TODO-
func (r *Gateway) GetWallets(id int64, reply *mutils.Wallets) error {
	var item mutils.WalletItem

	var count int

	rows, err := r.db.Query(
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
		gutils.RemoteLog.PutErrorS("query", "%v", err)

		return err
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
			&item.KeyID,
			&item.Signature,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "%v", err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "%v", err)

		return err
	}

	return err
}

//GetUsers -TODO-
func (r *Gateway) GetUsers(id int64, reply *mutils.Users) error {
	var item mutils.UserItem

	var count int

	rows, err := r.db.Query("SELECT user_id, coin_id FROM finances_user_transaction GROUP BY user_id, coin_id")
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "%v", err)

		return err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(
			&item.UserID,
			&item.CoinID,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("scan", "%v", err)

			return err
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorN("fetch", "%v", err)

		return err
	}

	return err
}

//GetWalletByID --TODO--
func (r *Gateway) GetWalletByID(id int64, reply *mutils.WalletItem) error {
	err := r.db.QueryRow(
		"SELECT "+
			"id,"+
			"name,"+
			"wallet,"+
			"date_added,"+
			"date_modified,"+
			"date_name_update,"+
			"coin_id,"+
			"user_id,"+
			"is_active,"+
			"is_deleted, "+
			"key_id,"+
			"signature "+
			"FROM users_userwallets WHERE id = ?", id).Scan(
		&reply.ID,
		&reply.Name,
		&reply.Wallet,
		&reply.DateAdded,
		&reply.DateModified,
		&reply.DateName,
		&reply.CoinID,
		&reply.UserID,
		&reply.IsActive,
		&reply.IsDeleted,
		&reply.KeyID,
		&reply.Signature,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", id, err)

		return err
	}

	return nil
}

//UpdateSignature -TODO-
func (r *Gateway) UpdateSignature(args mutils.ArgsUpdateSignature, reply *string) error {
	_, err := r.db.Exec("UPDATE "+args.Table+" SET key_id = ?, signature = ? WHERE ID = ?", args.KeyID, args.Signature, args.ID)

	if err != nil {
		gutils.RemoteLog.PutErrorS("exec", "(%s, %d), %v", args.Table, args.ID, err)

		return err
	}

	return nil
}

//GetSendingCount -TODO-
func (r *Gateway) GetSendingCount(id int64, reply *int64) error {
	err := r.db.QueryRow("SELECT COUNT(id) FROM finances_withdrawal_transaction WHERE coin_id = ? AND status = 'sending'", id).Scan(reply)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d), %v", id, err)

		return err
	}

	return nil
}

//GetWithdrawalTransactionsByStatusAndCoin -TODO-
func (r *Gateway) GetWithdrawalTransactionsByStatusAndCoin(args mutils.ArgsUCIB, reply *mutils.Withdrawals) error {
	var item mutils.WithdrawalItem

	var count int

	rows, err := r.db.Query(
		"SELECT "+
			"id, "+
			"parent_id, "+
			"status, "+
			"blockchain_transaction_id, "+
			"date_added, "+
			"date_confirmed, "+
			"coin_id, "+
			"user_id, "+
			"wallet_id, "+
			"reward_type, "+
			"amount, "+
			"debit, "+
			"tx_fee, "+
			"fee "+
			"FROM finances_withdrawal_transaction "+
			"WHERE AND coin_id = ? AND status = ?"+
			"ORDER BY ID",
		args.CoinID,
		args.Status,
	)
	if err != nil {
		gutils.RemoteLog.PutErrorS("query", "(%d, %s), %v", args.CoinID, args.Status, err)

		return err
	}
	defer rows.Close()

	var sqlTime mysql.NullTime

	for rows.Next() {
		err := rows.Scan(
			&item.ID,
			&item.ParentID,
			&item.Status,
			&item.Hash,
			&item.DateAdded,
			&sqlTime,
			&item.CoinID,
			&item.UserID,
			&item.WalletID,
			&item.RewardType,
			&item.Amount,
			&item.Debit,
			&item.TxFee,
			&item.Fee,
		)

		if err != nil {
			gutils.RemoteLog.PutErrorS("fetch", "(%d, %d, %s), %v", item.ID, args.CoinID, args.Status, err)

			return err
		}

		if sqlTime.Valid {
			item.DateConfirmed = sqlTime.Time
		} else {
			item.DateConfirmed = time.Time{}
		}

		item.Signatures = make(map[string]mutils.SignatureItem)

		r.GetWithdrawalSignaturesByID(item.ID, &item.Signatures)
		if err != nil {
			gutils.RemoteLog.PutErrorS("GetWithdrawalSignaturesByID", "%v", err)
		}

		*reply = append(*reply, item)

		count++
	}
	if err = rows.Err(); err != nil {
		gutils.RemoteLog.PutErrorS("fetch", "(%d, %s), %v", args.CoinID, args.Status, err)

		return err
	}

	return err
}
