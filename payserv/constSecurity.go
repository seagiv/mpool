package main

const sinRootCA = "rootCA.CRT"
const sinTLSImCA = "imCA.TLS.CRT"
const sinPayServCert = "PayServ.SVR.CRT"
const sinPayServKey = "PayServ.SVR.KEY"
const sinAdminCert = "Admin.SVR.CRT"
const sinAdminKey = "Admin.SVR.KEY"

const sinTLSCert = "PayServ.TLS.CRT"
const sinTLSKey = "PayServ.TLS.KEY"

const sinParams = "Params.JSON"

const statusProcessing = "processing"
const statusPending = "pending"
const statusApproved = "approved"
const statusProcessed = "processed"
const statusDeclined = "declined"
const statusSent = "sent"
const statusSending = "sending"
const statusSendError = "send_error"
const statusSendRetry = "send_retry"
const statusError = "error"

const connectPayServ = "connectPayServ"
const connectUtils = "connectUtils"
const connectCoins = "connectCoins"
const connectWallets = "connectWallets"
const connectCheck = "connectCheck"
const connectCorrection = "connectCorrection"
const connectDebug = "connectDebug"

const tableIncomesN = "table_incomes_n"     //normal records
const tableIncomesC = "table_incomes_c"     //corrections
const tableProxyPoolN = "table_proxypool_n" //normal records
const tableProxyPoolC = "table_proxypool_c" //corrections
const tableUserCPs = "table_usercps"
const tableCoinCPs = "table_coincps"
const tableCoins = "table_coins"
const tableWallets = "table_wallets"

const funcProcess = "func_Process"
const funcApprove = "func_Approve"
const funcDecline = "func_Decline"
const funcSend = "func_Send"
const funcCheck = "func_Check"
const funcSetStatus = "func_SetStatus"

const funcSign = "func_Sign"
const funcCoins = "func_Coins"
const funcWallets = "func_Wallets"
const funcApply = "func_Apply"

const funcEnable = "func_Enable"

const cnAdmin = "SVR Admin"
const cnAutoPay = "SVR AutoPay"
const cnPayServ = "SVR PayServ"

const cntlsDBGW = "DBGW"
const cntlsPSCli = "PSCli"
const cntlsBlind = "blindenhund"
const cntlsAdmin = "admin"

var allowedCNs = map[string][]string{
	statusProcessing: {cnAutoPay},
	statusPending:    {cnPayServ},
	statusApproved:   {cnPayServ},
	statusProcessed:  {cnPayServ},
	statusDeclined:   {cnPayServ},
	statusSending:    {cnPayServ},
	statusSent:       {cnPayServ},
	statusSendError:  {cnPayServ},
	statusSendRetry:  {cnPayServ},

	tableUserCPs:    {cnPayServ},
	tableCoinCPs:    {cnPayServ},
	tableIncomesN:   {cnAdmin},
	tableIncomesC:   {cnPayServ},
	tableProxyPoolN: {cnAdmin},
	tableProxyPoolC: {cnPayServ},
	tableCoins:      {cnAdmin},
	tableWallets:    {cnAdmin},

	funcProcess:   {cntlsBlind, cntlsPSCli},
	funcApprove:   {cntlsBlind, cntlsPSCli},
	funcDecline:   {cntlsBlind, cntlsPSCli},
	funcSend:      {cntlsBlind, cntlsPSCli},
	funcCheck:     {cntlsBlind, cntlsPSCli},
	funcSetStatus: {cntlsBlind, cntlsPSCli},
	funcEnable:    {cntlsPSCli},

	connectPayServ:    {cntlsBlind, cntlsPSCli},
	connectUtils:      {cntlsBlind, cntlsPSCli},
	connectCoins:      {cntlsPSCli},
	connectWallets:    {cntlsPSCli},
	connectCheck:      {cntlsPSCli},
	connectCorrection: {cntlsPSCli},
	connectDebug:      {cntlsPSCli},
}
