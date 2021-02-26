package main

import (
	"github.com/seagiv/common/coinapi"
	"github.com/seagiv/common/gutils"
	"github.com/seagiv/mpool/mutils"
)

//Debug --TODO--
type Debug struct {
	сlientCN string
}

func (d *Debug) onConnect(commonName string) bool {
	d.сlientCN = commonName

	return gutils.IsIn(commonName, allowedCNs[connectDebug])
}

func (d *Debug) onDisconnect() {
}

//SendTo -
func (d *Debug) SendTo(args mutils.ArgsDST, txHash *string) error {
	var err error

	//coinAPI
	apiCoin, err := coinapi.GetCoinAPI(0, args.CoinTag)
	if err != nil {
		return gutils.RemoteLog.PutErrorS("GetCoinAPI", "%v", err)
	}

	apiCoin.SetServiceAccount(args.AddressFrom, args.PrivateKey)

	txHash, _, _, err = apiCoin.Send(args.Amount, args.AddressTo)
	if err != nil {
		return gutils.RemoteLog.PutErrorS("Send", "%v", err)
	}

	return nil
}
