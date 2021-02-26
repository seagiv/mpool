package main

import (
	"fmt"
	"strings"

	"github.com/seagiv/common/coinapi"
	"github.com/seagiv/common/gutils"
)

//Utils --TODO--
type Utils struct {
	сlientCN string
}

func (u *Utils) onConnect(commonName string) bool {
	u.сlientCN = commonName

	return gutils.IsIn(commonName, allowedCNs[connectUtils])
}

func (u *Utils) onDisconnect() {
}

//Ping -TODO-
func (u *Utils) Ping(id int64, ack *bool) error {
	*ack = true

	return nil
}

//IsProcessing -TODO-
func (u *Utils) IsProcessing(id int64, ack *bool) error {
	*ack = txLocker.Get(getLockNameTx(id)) != nil

	return nil
}

//ListLocks -TODO-
func (u *Utils) ListLocks(id int64, ack *string) error {
	*ack = txLocker.List()

	return nil
}

//IsSendAllowed -TODO-
func (u *Utils) IsSendAllowed(coinID int64, ack *bool) error {
	*ack = coinapi.Coins[IDToTag(coinID)] != nil

	return nil
}

//GetCoinAPI -TODO-
func (u *Utils) GetCoinAPI(coinID int64, ack *string) error {
	api, err := coinapi.GetCoinAPI(0, IDToTag(coinID))

	if err != nil {
		*ack = ""
	} else {
		apt := strings.Split(fmt.Sprintf("%T", api), ".")

		if len(apt) > 0 {
			*ack = apt[1]
		}
	}

	return nil
}

//Enabled -TODO-
func (u *Utils) Enabled(status bool, ack *string) error {
	if !gutils.IsIn(u.сlientCN, allowedCNs[funcEnable]) {
		return gutils.FormatErrorS("commonName", "access DENIED to function %s for client [%s]", funcEnable, u.сlientCN)
	}

	enabledPayServ = status

	if status {
		gutils.RemoteLog.PutInfoN("PayServ - ENABLED")
	} else {
		gutils.RemoteLog.PutInfoN("PayServ - DISABLED")
	}

	*ack = "OK"

	return nil
}
