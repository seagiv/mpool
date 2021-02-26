package mutils

import (
	"fmt"
	"time"

	"github.com/seagiv/foreign/decimal"
)

//ArgsUCIB set of params used in number of RCP calls
type ArgsUCIB struct {
	UserID int64
	CoinID int64
	FromID int64
	Limit  int64
	Status string
	Hash   string
}

//ArgsUpdateSignature -TODO-
type ArgsUpdateSignature struct {
	ID int64

	Table string

	Signature []byte
	KeyID     []byte
}

//ArgsUpdateWithdrawal --TODO--
type ArgsUpdateWithdrawal struct {
	ID int64

	UserID int64
	CoinID int64

	Status  string
	Comment string
	Hash    string

	Confirmed time.Time

	TxFee decimal.Decimal

	Signature []byte
	KeyID     []byte
}

//ArgsCorrection -TODO-
type ArgsCorrection struct {
	UserID int64
	CoinID int64
	PoolID int64
	Amount decimal.Decimal
}

//ArgsIS -TODO-
type ArgsIS struct {
	ID int64
	S  string
}

//SendResult -TODO-
type SendResult struct {
	ID      int64
	Result  string
	Comment string
}

//SendResults -TODO-
type SendResults []SendResult

//ReplyCorrection -TODO-
type ReplyCorrection struct {
	PoolID   int64
	IncomeID int64
}

//SignatureItem -TODO-
type SignatureItem struct {
	ID int64

	TimeStamp time.Time

	Signature []byte
	KeyID     []byte
}

//WithdrawalItem -TODO-
type WithdrawalItem struct {
	ID       int64
	ParentID int64

	Status string
	Hash   string

	Comment string

	DateAdded     time.Time
	DateConfirmed time.Time

	CoinID   int64
	UserID   int64
	WalletID int64

	ModifiedByID int64

	RewardType int64

	Amount decimal.Decimal
	Debit  decimal.Decimal
	Fee    decimal.Decimal

	TxFee decimal.Decimal

	Address string

	Signatures map[string]SignatureItem
}

//Withdrawals -TODO-
type Withdrawals []WithdrawalItem

//IncomeItem -TODO-
type IncomeItem struct {
	ID int64

	Status    int
	BlockHash string

	Amount  decimal.Decimal
	Balance decimal.Decimal

	DateAdded    time.Time
	DateModified time.Time

	PoolID int64
	UserID int64
	CoinID int64

	TypeID int64

	Signature []byte
	KeyID     []byte
}

//Incomes -TODO-
type Incomes []IncomeItem

//UserCPItem -TODO-
type UserCPItem struct {
	ID int64

	LastBalanceID    int64 //ID for finances_user_transaction !!!!!
	LastWithdrawalID int64

	LastBlockHash string

	UserID int64
	CoinID int64

	TimeStamp time.Time

	CreditAll  decimal.Decimal
	CreditUser decimal.Decimal
	Debit      decimal.Decimal
	SentAll    decimal.Decimal

	Signature []byte
	KeyID     []byte
}

//UserCPs -TODO-
type UserCPs []UserCPItem

//ReplyUserCPVerification -TODO-
type ReplyUserCPVerification struct {
	LastBalanceID    int64
	LastWithdrawalID int64

	CreditAll  decimal.Decimal
	CreditUser decimal.Decimal
	Debit      decimal.Decimal
	SentAll    decimal.Decimal
}

//CoinCPItem -TODO-
type CoinCPItem struct {
	ID int64

	CoinID int64

	LastPPID        int64
	LastPPBalance   decimal.Decimal
	LastPPBlockHash string

	LastIncomeID        int64
	LastIncomeFee       decimal.Decimal
	LastIncomeBlockHash string

	TimeStamp time.Time

	Signature []byte
	KeyID     []byte
}

//CoinCPs -TODO-
type CoinCPs []CoinCPItem

//ProxyPoolItemCombined -TODO-
type ProxyPoolItemCombined struct {
	ID int64

	DateAdded time.Time

	CoinID int64
	PoolID int64

	TypeID int64

	RewardTypeID int64

	BlockHash string

	AmountPPT decimal.Decimal
	AmountUBT decimal.Decimal

	Balance decimal.Decimal

	TransactionDate time.Time

	PoolBalance decimal.Decimal

	Confirmed int64

	Signature []byte
	KeyID     []byte
}

//ProxyPoolsCombined -TODO-
type ProxyPoolsCombined []ProxyPoolItemCombined

//CoinItem -TODO-
type CoinItem struct {
	ID int64

	DateAdded    time.Time
	DateModified time.Time

	Tag  string
	Name string

	IsActive bool

	AlgorithmID int64
	Port        int
	RewardType  int64

	FeeOur float64

	Confirmations int64

	Description string

	Icon string

	IsViewable bool

	OrphanReward     decimal.Decimal
	FeeWithdrawal    decimal.Decimal
	MinSumWithdrawal decimal.Decimal

	Signature []byte
	KeyID     []byte
}

//Coins -TODO-
type Coins []CoinItem

//WalletItem -TODO-
type WalletItem struct {
	ID           int64
	Name         string
	Wallet       string
	DateAdded    time.Time
	DateModified time.Time
	DateName     time.Time
	UserID       int64
	CoinID       int64
	IsActive     bool
	IsDeleted    bool

	Signature []byte
	KeyID     []byte
}

//Wallets -TODO-
type Wallets []WalletItem

//UserItem -TODO-
type UserItem struct {
	UserID int64
	CoinID int64
}

//Users -TODO-
type Users []UserItem

//GhostItem -TODO-
type GhostItem struct {
	ID        int64
	BlockHash string
}

//Ghosts -TODO-
type Ghosts []GhostItem

//ArgsDST args for rpc calls
type ArgsDST struct {
	CoinTag     string
	AddressFrom string
	AddressTo   string
	PrivateKey  string
	Amount      decimal.Decimal
}

//GetWithdrawalSignature -TODO-
func GetWithdrawalSignature(w *WithdrawalItem) string {
	return fmt.Sprintf("%d%d%s%d%d%d%d%s%s%s%d%s",
		w.ID,
		w.ParentID,
		w.Status,
		w.UserID,
		w.CoinID,
		w.WalletID,
		w.RewardType,
		w.Amount.String(),
		w.Debit.String(),
		w.Fee.String(),
		w.DateAdded.UTC().Unix(),

		w.Hash,
	)
}

//GetUserCPSignature -TODO-
func GetUserCPSignature(c *UserCPItem) string {
	return fmt.Sprintf("%d%d%d%d%s%s%s%s%s%d",
		c.UserID,
		c.CoinID,
		c.LastBalanceID,
		c.LastWithdrawalID,
		c.LastBlockHash,
		c.CreditAll.String(),
		c.CreditUser.String(),
		c.Debit.String(),
		c.SentAll.String(),
		c.TimeStamp.Unix(),
	)
}

//GetCoinCPSignature -TODO-
func GetCoinCPSignature(c *CoinCPItem) string {
	return fmt.Sprintf("%d%d%s%s%d%s%s%d",
		c.CoinID,
		c.LastPPID,
		c.LastPPBalance.String(),
		c.LastPPBlockHash,
		c.LastIncomeID,
		c.LastIncomeFee.String(),
		c.LastIncomeBlockHash,
		c.TimeStamp.Unix(),
	)
}

//GetIncomeSignature -TODO-
func GetIncomeSignature(u *IncomeItem) string {
	return fmt.Sprintf("%d%d%d%d%d%s%s%s%d",
		u.PoolID,
		u.CoinID,
		u.UserID,
		u.TypeID,
		u.Status,
		u.BlockHash,
		u.Amount.String(),
		u.Balance.String(),
		u.DateAdded.Unix(),
	)
}

//GetProxyPoolSignature -TODO-
func GetProxyPoolSignature(pp *ProxyPoolItemCombined) string {
	return fmt.Sprintf("%d%d%d%d%d%s%s%s%d%d",
		pp.ID,
		pp.PoolID,
		pp.CoinID,
		pp.TypeID,
		pp.RewardTypeID,
		pp.BlockHash,
		pp.AmountPPT.String(),
		pp.Balance.String(),
		pp.Confirmed,
		pp.DateAdded.UTC().Unix(),
	)
}

//GetCoinSignature -TODO-
func GetCoinSignature(c *CoinItem) string {
	return fmt.Sprintf("%d%d%s%d%d%f%d%s%s%s",
		c.ID,
		c.DateAdded.UTC().Unix(),
		c.Tag,
		c.AlgorithmID,
		c.RewardType,
		c.FeeOur,
		c.Confirmations,
		c.OrphanReward.String(),
		c.FeeWithdrawal.String(),
		c.MinSumWithdrawal.String(),
	)
}

//GetWalletSignature -TODO-
func GetWalletSignature(w *WalletItem) string {
	return fmt.Sprintf("%d%s%d%d%d%d",
		w.ID,
		//w.Name,
		w.Wallet,
		w.DateAdded.UTC().Unix(),
		w.DateModified.UTC().Unix(),
		w.UserID,
		w.CoinID,
	)
}
