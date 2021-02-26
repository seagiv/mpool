package main

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"sync"

	"github.com/seagiv/common/gutils"
	"github.com/seagiv/mpool/mutils"
)

const fnUserCP = "/var/lib/payserv/checkpoints.user"

var storageUserCP *UserCPStorage

var prefixUserMutex = "usercp_"

//UserCPStorage -TODO-
type UserCPStorage struct {
	sync.Mutex
	Data map[string]*mutils.UserCPItem
}

//NewUserCPStorage -TODO-
func NewUserCPStorage() *UserCPStorage {
	cp := UserCPStorage{Data: make(map[string]*mutils.UserCPItem)}

	cp.load()

	return &cp
}

//Load -TODO-
func (s *UserCPStorage) load() error {
	var err error

	//gutils.RemoteLog.PutInfoN("loading cache from %s", fnUserCP)

	err = gutils.LoadObject(fnUserCP, &s.Data)
	if err != nil {
		gutils.RemoteLog.PutWarningN("USER-CP CACHE (%s) NOT FOUND", fnUserCP)
	} else {
		gutils.RemoteLog.PutInfoN("%d coin checkpoint(s) found in cache (%s)", len(s.Data), fnCoinCP)
	}

	return err
}

//Get -TODO-
func (s *UserCPStorage) Get(userID, coinID int64) *mutils.UserCPItem {
	s.Lock()
	cp := s.Data[fmt.Sprintf(prefixUserMutex+"%d_%d", userID, coinID)]
	defer s.Unlock()

	return cp
}

//Set -TODO-
func (s *UserCPStorage) Set(cp mutils.UserCPItem) {
	s.Lock()
	key := fmt.Sprintf(prefixUserMutex+"%d_%d", cp.UserID, cp.CoinID)

	s.Data[key] = &cp

	gutils.SaveObject(fnUserCP, s.Data)

	defer s.Unlock()
}

//Valid -TODO-
func (s *UserCPStorage) Valid(cpc mutils.UserCPItem) bool {
	cp := s.Get(cpc.UserID, cpc.CoinID)

	if cp == nil {
		s.Set(cpc)

		return true
	}

	return (bytes.Compare(cp.KeyID, cpc.KeyID) == 0) && (bytes.Compare(cp.Signature, cpc.Signature) == 0)
}

func (s *UserCPStorage) checkUserCPs(c *gutils.ClientConnection) error {
	var err error

	var replyCPs mutils.UserCPs

	err = c.Call("Gateway.GetUserCPs", 0, &replyCPs)
	if err != nil {
		return gutils.FormatErrorS("GetUserCPs", "error %v", err)
	}

	for i := 0; i < len(replyCPs); i++ {
		certA, err := gutils.GetCertificate(c, replyCPs[i].KeyID, allowedCNs[tableUserCPs], replyCPs[i].TimeStamp)
		if err != nil {
			return gutils.FormatErrorS("getCertificate", "error %v", err)
		}

		dataToCheck := mutils.GetUserCPSignature(&replyCPs[i])

		err = gutils.VerifySignature([]byte(dataToCheck), replyCPs[i].Signature, certA.PublicKey.(*rsa.PublicKey))
		if err != nil {
			return gutils.FormatErrorS("VerifySignature", "(%d, %d) error %v", replyCPs[i].UserID, replyCPs[i].CoinID, err)
		}

		if !s.Valid(replyCPs[i]) {
			return gutils.FormatErrorS("IsValid", "userCP(%d, %d) mismatch with cached ones", replyCPs[i].UserID, replyCPs[i].CoinID)
		}
	}

	if len(s.Data) > 0 {
		gutils.RemoteLog.PutInfoN("%d userCPs synced between database and local cache", len(s.Data))
	}

	return nil
}
