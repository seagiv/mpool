package main

import (
	"encoding/json"

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

//Version -TODO-
func (u *Utils) Version(id int64, reply *gutils.ReplyVersion) error {
	(*reply).Project = projectInfo
	(*reply).Version = buildInfo

	return nil
}

//Params -TODO-
type Params struct {
	DataSrouce string `json:"datasource"`
}

//SetParams -TODO-
func (u *Utils) SetParams(params string, reply *string) error {
	var err error
	var p Params

	err = json.Unmarshal([]byte(params), &p)
	if err != nil {
		return gutils.FormatErrorS("json.Unmarshal", "%v", err)
	}

	dataSource = p.DataSrouce

	return err
}
