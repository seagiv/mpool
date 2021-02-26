package main

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"sync"

	"github.com/seagiv/common/gutils"
	"github.com/seagiv/mpool/mutils"
)

const fnCoinCP = "/var/lib/payserv/checkpoints.coin"

var storageCoinCP *CoinCPStorage

var prefixCoinMutex = "coincp_"

//CoinCPStorage -TODO-
type CoinCPStorage struct {
	sync.Mutex
	Data map[string]*mutils.CoinCPItem
}

//NewCoinCPStorage -TODO-
func NewCoinCPStorage() *CoinCPStorage {
	cp := CoinCPStorage{Data: make(map[string]*mutils.CoinCPItem)}

	cp.load()

	return &cp
}

func (s *CoinCPStorage) load() error {
	var err error

	//gutils.RemoteLog.PutInfoN("loading cache from %s", fnCoinCP)

	err = gutils.LoadObject(fnCoinCP, &s.Data)
	if err != nil {
		gutils.RemoteLog.PutWarningN("COIN-CP CACHE (%s) NOT FOUND", fnCoinCP)
	} else {
		gutils.RemoteLog.PutInfoN("%d coin checkpoint(s) found in cache (%s)", len(s.Data), fnCoinCP)
	}

	return err
}

//Get -TODO-
func (s *CoinCPStorage) Get(coinID int64) *mutils.CoinCPItem {
	s.Lock()
	cp := s.Data[fmt.Sprintf(prefixCoinMutex+"%d", coinID)]
	defer s.Unlock()

	return cp
}

//Set -TODO-
func (s *CoinCPStorage) Set(cp mutils.CoinCPItem) {
	s.Lock()
	key := fmt.Sprintf(prefixCoinMutex+"%d", cp.CoinID)

	s.Data[key] = &cp

	gutils.SaveObject(fnCoinCP, s.Data)

	defer s.Unlock()
}

//Valid -TODO-
func (s *CoinCPStorage) Valid(cpc mutils.CoinCPItem) bool {
	cp := s.Get(cpc.CoinID)

	if cp == nil {
		s.Set(cpc)

		return true
	}

	return (bytes.Compare(cp.KeyID, cpc.KeyID) == 0) && (bytes.Compare(cp.Signature, cpc.Signature) == 0)
}

func (s *CoinCPStorage) checkCoinCPs(c *gutils.ClientConnection) error {
	var err error

	var replyCPs mutils.CoinCPs

	err = c.Call("Gateway.GetCoinCPs", 0, &replyCPs)
	if err != nil {
		return gutils.FormatErrorS("GetCoinCPs", "error %v", err)
	}

	for i := 0; i < len(replyCPs); i++ {
		certA, err := gutils.GetCertificate(c, replyCPs[i].KeyID, allowedCNs[tableCoinCPs], replyCPs[i].TimeStamp)
		if err != nil {
			return gutils.FormatErrorS("getCertificate", "error %v", err)
		}

		dataToCheck := mutils.GetCoinCPSignature(&replyCPs[i])

		err = gutils.VerifySignature([]byte(dataToCheck), replyCPs[i].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return gutils.FormatErrorS("VerifySignature", "(%d) error %v", replyCPs[i].CoinID, err)
		}

		if !s.Valid(replyCPs[i]) {
			return gutils.FormatErrorS("IsValid", "coinCP(%d) mismatch with cached ones", replyCPs[i].CoinID)
		}
	}

	if len(s.Data) > 0 {
		gutils.RemoteLog.PutInfoN("%d coinCPs synced between database and local cache", len(s.Data))
	}

	return nil
}
