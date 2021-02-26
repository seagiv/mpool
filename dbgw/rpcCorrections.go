package main

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/seagiv/common/gutils"
	"github.com/seagiv/foreign/decimal"
	"github.com/seagiv/mpool/mutils"
)

const ttCorrection = 13

//Corrections --TODO--
type Corrections struct {
	сlientCN string

	Tx *sql.Tx
	Wd *time.Timer

	db *sql.DB
}

func (c *Corrections) onConnect(commonName string) bool {
	var err error

	c.сlientCN = commonName

	if !gutils.IsIn(commonName, allowedCNs[connectCorrections]) {
		return false
	}

	if len(dataSource) == 0 {
		return false
	}

	c.db, err = sql.Open("mysql", dataSource)
	if err != nil {
		gutils.RemoteLog.PutErrorN("sql.Open error %s", err.Error())

		return false
	}

	return true
}

func (c *Corrections) onDisconnect() {
	//gutils.RemoteLog.PutInfo("Corrections.onDisconnect")

	if c.Tx != nil {
		if c.Wd != nil {
			c.Wd.Stop()
		}

		err := c.Tx.Rollback()
		if err == nil {
			gutils.RemoteLog.PutDebugN("rolled back unfinished transaction.")
		}
	}

	if c.db != nil {
		c.db.Close()
	}
}

func (c *Corrections) watchdog() {
	if c.Tx != nil {
		gutils.RemoteLog.PutDebugN("correction transaction took too long, rolling it back.")

		c.Tx.Rollback()
	}
}

//SetLock --TODO--
func (c *Corrections) SetLock(args mutils.ArgsUCIB, reply *string) error {
	var err error

	*reply = "-"

	c.Tx, err = c.db.Begin()
	if err != nil {
		return gutils.FormatErrorS("begin", "%v", err)
	}

	c.Wd = time.AfterFunc(30*time.Second, c.watchdog)

	//creating table locks
	_, err = c.Tx.Exec("SELECT id FROM finances_proxypool_transactions WHERE coin_id = ? FOR UPDATE", args.CoinID)
	if err != nil {
		return gutils.FormatErrorF("lock", 0, "proxypool", "%v", err)
	}

	_, err = c.Tx.Exec("SELECT id FROM finances_user_transaction WHERE user_id = ? AND coin_id = ? FOR UPDATE", args.UserID, args.CoinID)
	if err != nil {
		return gutils.FormatErrorF("lock", 0, "income", "%v", err)
	}

	var id int64

	err = c.Tx.QueryRow("SELECT id FROM finances_proxypool_transactions WHERE is_calculated = 0 AND is_confirmed IN (0, 1, 3) AND coin_id = ?",
		args.CoinID,
	).Scan(
		&id,
	)
	if err == sql.ErrNoRows {
		*reply = "+"
	} else {
		c.Wd.Stop()

		c.Tx.Rollback()
	}

	return nil
}

//Apply --TODO--
func (c *Corrections) Apply(args mutils.ArgsCorrection, reply *mutils.ReplyCorrection) error {
	var err error

	if c.Tx == nil {
		return gutils.FormatErrorF("lock", 0, "check", "must call CorrectionSetLock first")
	}

	//updating finances_proxypool_transactions
	var prevAmount decimal.NullDecimal
	var prevBalance decimal.NullDecimal

	err = c.Tx.QueryRow("SELECT amount, balance FROM finances_proxypool_transactions "+
		"WHERE transaction_type_id IN (1,13) AND is_confirmed IN (1,3) AND is_calculated = 1 AND coin_id = ? ORDER BY id DESC LIMIT 1 FOR UPDATE",
		args.CoinID,
	).Scan(
		&prevAmount,
		&prevBalance,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			return gutils.FormatErrorF("query", 0, "proxypool, balance", "%v", err)
		}

		prevAmount.Decimal = decimal.Zero
		prevBalance.Decimal = decimal.Zero
	}

	var itemPP mutils.ProxyPoolItemCombined

	itemPP.PoolID = args.PoolID
	itemPP.CoinID = args.CoinID
	itemPP.TypeID = ttCorrection
	itemPP.BlockHash = "correction_v2_"
	itemPP.RewardTypeID = 1
	itemPP.AmountPPT = args.Amount
	itemPP.Balance = prevBalance.Decimal.Add(prevAmount.Decimal)
	itemPP.Confirmed = 1
	itemPP.DateAdded = time.Now()

	var prevPoolAmount decimal.NullDecimal
	var prevPoolBalance decimal.NullDecimal

	err = c.Tx.QueryRow("SELECT amount, pool_balance FROM finances_proxypool_transactions "+
		"WHERE transaction_type_id IN (1,13) AND is_confirmed IN (1,3) AND is_calculated = 1 AND coin_id = ? AND pool_id = ? ORDER BY id DESC LIMIT 1 FOR UPDATE",
		args.CoinID,
		args.PoolID,
	).Scan(
		&prevPoolAmount,
		&prevPoolBalance,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			return gutils.FormatErrorF("query", 0, "proxypool, pool_balance", "%v", err)
		}

		prevPoolAmount.Decimal = decimal.Zero
		prevPoolBalance.Decimal = decimal.Zero
	}

	itemPP.PoolBalance = prevPoolBalance.Decimal.Add(prevPoolAmount.Decimal)

	result, err := c.Tx.Exec("INSERT INTO finances_proxypool_transactions("+
		"date_added, "+
		"amount, "+
		"coin_id, "+
		"pool_id, "+
		"transaction_type_id, "+
		"blockhash, "+
		"transaction_date, "+
		"reward_type, "+
		"balance, "+
		"pool_balance, "+
		"is_calculated, "+
		"is_confirmed"+
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		itemPP.DateAdded,
		itemPP.AmountPPT,
		itemPP.CoinID,
		itemPP.PoolID,
		itemPP.TypeID,
		itemPP.BlockHash,
		itemPP.DateAdded,
		itemPP.RewardTypeID,
		itemPP.Balance,
		itemPP.PoolBalance,
		true,
		itemPP.Confirmed,
	)
	if err != nil {
		return gutils.FormatErrorF("insert", 0, "proxypool", "%v", err)
	}

	reply.PoolID, err = result.LastInsertId()
	if err != nil {
		return gutils.FormatErrorF("insert", 0, "proxypool, id", "%v", err)
	}

	itemPP.BlockHash = itemPP.BlockHash + strconv.FormatInt(reply.PoolID, 10)

	_, err = c.Tx.Exec("UPDATE finances_proxypool_transactions SET blockhash = ? WHERE id = ?", itemPP.BlockHash, reply.PoolID)
	if err != nil {
		return gutils.FormatErrorF("update", 0, "proxypool", "%v", err)
	}

	//updating finances_user_transaction
	err = c.Tx.QueryRow("SELECT amount, balance FROM finances_user_transaction WHERE user_id = ? AND coin_id = ? ORDER BY id DESC LIMIT 1 FOR UPDATE",
		args.UserID,
		args.CoinID,
	).Scan(
		&prevAmount,
		&prevBalance,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			return gutils.FormatErrorF("query", 0, "income, balance", "%v", err)
		}

		prevAmount.Decimal = decimal.Zero
		prevBalance.Decimal = decimal.Zero
	}

	var itemUT mutils.IncomeItem

	itemUT.PoolID = args.PoolID
	itemUT.CoinID = args.CoinID
	itemUT.UserID = args.UserID
	itemUT.TypeID = ttCorrection
	itemUT.Status = 3
	itemUT.BlockHash = itemPP.BlockHash
	itemUT.Amount = args.Amount
	itemUT.Balance = prevBalance.Decimal.Add(prevAmount.Decimal)
	itemUT.DateAdded = time.Now()

	result, err = c.Tx.Exec("INSERT INTO finances_user_transaction("+
		"date_added, "+
		"amount, "+
		"status, "+
		"user_id, "+
		"coin_id, "+
		"pool_id, "+
		"transaction_type_id, "+
		"blockhash, "+
		"date_modified, "+
		"balance "+
		") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		itemUT.DateAdded,
		itemUT.Amount,
		itemUT.Status,
		itemUT.UserID,
		itemUT.CoinID,
		itemUT.PoolID,
		itemUT.TypeID,
		itemUT.BlockHash,
		itemUT.DateAdded,
		itemUT.Balance,
	)
	if err != nil {
		return gutils.FormatErrorF("insert", 0, "income", "%v", err)
	}

	reply.IncomeID, err = result.LastInsertId()
	if err != nil {
		return gutils.FormatErrorF("insert", 0, "income, id", "%v", err)
	}

	return nil
}

//GetPPByID -TODO-
func (c *Corrections) GetPPByID(id int64, reply *mutils.ProxyPoolItemCombined) error {
	if c.Tx == nil {
		return gutils.FormatErrorF("lock", 0, "check", "must call CorrectionSetLock first")
	}

	err := c.Tx.QueryRow(
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
			"FROM finances_proxypool_transactions WHERE id = ? FOR UPDATE",
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

		return err
	}

	return err
}

//GetUTByID --TODO--
func (c *Corrections) GetUTByID(id int64, reply *mutils.IncomeItem) error {
	if c.Tx == nil {
		return gutils.FormatErrorF("lock", 0, "check", "must call CorrectionSetLock first")
	}

	err := c.Tx.QueryRow(
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
			"FROM finances_user_transaction WHERE ID = ? FOR UPDATE",
		id,
	).Scan(
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

//UpdateSignature --TODO--
func (c *Corrections) UpdateSignature(args mutils.ArgsUpdateSignature, reply *string) error {
	if c.Tx == nil {
		return gutils.FormatErrorF("lock", 0, "check", "must call CorrectionSetLock first")
	}

	_, err := c.Tx.Exec("UPDATE "+args.Table+" SET key_id = ?, signature = ? WHERE ID = ?", args.KeyID, args.Signature, args.ID)

	if err != nil {
		gutils.RemoteLog.PutErrorN("(%s, %d).Exec error %v", args.Table, args.ID, err)

		return err
	}

	return nil
}

//Commit --TODO--
func (c *Corrections) Commit(id int64, reply *string) error {
	if c.Tx == nil {
		return gutils.FormatErrorF("lock", 0, "check", "must call CorrectionSetLock first")
	}

	c.Wd.Stop()

	err := c.Tx.Commit()
	if err != nil {
		gutils.RemoteLog.PutErrorN("%v", err)

		return err
	}

	c.Tx = nil

	return nil
}
