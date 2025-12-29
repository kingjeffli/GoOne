package gconf

import (
	"fmt"
	"github.com/Iori372552686/GoOne/lib/util/marshal"
	"reflect"
	"testing"
)

func debugVariable(variable interface{}) string {
	val := reflect.ValueOf(variable)
	typ := reflect.TypeOf(variable)

	ret := fmt.Sprintf("%s: %v", typ, val)
	return ret
}

/*
if error
t.Error()
or
t.Errorf()
*/
func TestLoad(t *testing.T) {
	//var confFile = flag.String("svr_conf", "./server_conf_ide.yaml", "app config file")
	var confFile = "./server_conf_ide.yaml"

	fmt.Println("-----------------------------------------connConfig------------")
	// load connsvr
	var connCfg = &ConnConfig{}
	err := marshal.LoadConfFile(confFile, &connCfg)
	if err != nil {
		t.Error(err)
	}
	if connCfg.SelfBusId != "1.1.1.2" {
		t.Error(debugVariable(connCfg.SelfBusId))
	}
	if connCfg.BusMQAddr != "amqp://guest:guest@192.168.50.250:5672/" {
		t.Error(debugVariable(connCfg.BusMQAddr))
	}
	if connCfg.RegisterAddr != "192.168.50.250:2182" {
		t.Error(debugVariable(connCfg.RegisterAddr))
	}
	if len(connCfg.HTTPSigns) != 0 {
		t.Error(debugVariable(connCfg.HTTPSigns))
	}
	if len(connCfg.RestApiConf) != 1 {
		t.Error(debugVariable(connCfg.RestApiConf))
	}
	if connCfg.ListenPort != 11000 {
		t.Error(debugVariable(connCfg.ListenPort))
	}

	fmt.Println("-----------------------------------------infoConfig------------")

	var infoCfg = &InfoConfig{}
	err = marshal.LoadConfFile(confFile, &infoCfg)
	if err != nil {
		t.Error(err)
	}

	if len(infoCfg.DbInstances) != 1 {
		t.Error(debugVariable(infoCfg.DbInstances))
	}
	if infoCfg.SelfBusId != "1.1.3.1" {
		t.Error(debugVariable(infoCfg.SelfBusId))
	}
	if infoCfg.BusMQAddr != "amqp://guest:guest@192.168.50.250:5672/" {
		t.Error(debugVariable(infoCfg.BusMQAddr))
	}
	if infoCfg.RegisterAddr != "192.168.50.250:2182" {
		t.Error(debugVariable(infoCfg.RegisterAddr))
	}
	if infoCfg.DbInstances[0].InstanceID != 3 {
		t.Error("infoCfg.DbInstances[0].InstanceID", infoCfg.DbInstances[0].InstanceID)
	}

	fmt.Println("-----------------------------------------MainSvrConfig------------")

	var mainCfg = &MainSvrConfig{}
	err = marshal.LoadConfFile(confFile, &mainCfg)
	if err != nil {
		t.Error(err)
	}

	if mainCfg.Pprof != true {
		t.Error(debugVariable(mainCfg.Pprof))
	}
	if mainCfg.SensitiveWordsFile != "../common/conf/sensitive.txt" {
		t.Error(debugVariable(mainCfg.SensitiveWordsFile))
	}
	if mainCfg.SelfBusId != "1.1.2.2" {
		t.Error(debugVariable(mainCfg.SelfBusId))
	}
	if mainCfg.BusMQAddr != "amqp://guest:guest@192.168.50.250:5672/" {
		t.Error(debugVariable(mainCfg.BusMQAddr))
	}
	if mainCfg.RegisterAddr != "192.168.50.250:2182" {
		t.Error(debugVariable(mainCfg.RegisterAddr))
	}
	if mainCfg.NacosConf.Port != 8848 {
		t.Error(debugVariable(mainCfg.NacosConf))
	}
	if mainCfg.GameDataDir != "../common/gamedata/data" {
		t.Error(debugVariable(mainCfg.GameDataDir))
	}
	if mainCfg.SensitiveWordsFile != "../common/conf/sensitive.txt" {
		t.Error("mainCfg.SensitiveWordsFile", mainCfg.SensitiveWordsFile)
	}

	fmt.Println("-----------------------------------------MySqlSvrConfig------------")

	var mysqlCfg = &MySqlSvrConfig{}
	err = marshal.LoadConfFile(confFile, &mysqlCfg)
	if err != nil {
		t.Error(err)
	}

	if mysqlCfg.SelfBusId != "1.1.4.2" {
		t.Error(debugVariable(mysqlCfg.SelfBusId))
	}
	if mysqlCfg.BusMQAddr != "amqp://guest:guest@192.168.50.250:5672/" {
		t.Error(debugVariable(mysqlCfg.BusMQAddr))
	}
	if mysqlCfg.RegisterAddr != "192.168.50.250:2182" {
		t.Error(debugVariable(mysqlCfg.RegisterAddr))
	}

	fmt.Println("------------------------------------------RoomCenterConfig------------")

	var roomCenterCfg = &RoomCenterConfig{}
	err = marshal.LoadConfFile(confFile, &roomCenterCfg)
	if err != nil {
		t.Error(err)
	}
	if roomCenterCfg.SelfBusId != "1.1.11.2" {
		t.Error(debugVariable(roomCenterCfg.SelfBusId))
	}
	if roomCenterCfg.BusMQAddr != "amqp://guest:guest@192.168.50.250:5672/" {
		t.Error(debugVariable(roomCenterCfg.BusMQAddr))
	}
	if roomCenterCfg.RegisterAddr != "192.168.50.250:2182" {
		t.Error(debugVariable(roomCenterCfg.RegisterAddr))
	}

	if roomCenterCfg.DbInstances[0].Description != "brief info data" {
		t.Error("roomCenterCfg.DbInstances[0].Description", roomCenterCfg.DbInstances[0].Description)
	}

	fmt.Println("------------------------------------------TexasConfig------------")

	var texasCfg = &TexasConfig{}
	err = marshal.LoadConfFile(confFile, &texasCfg)
	if err != nil {
		t.Error(err)
	}

	if texasCfg.Pprof != true {
		t.Error(debugVariable(texasCfg.Pprof))
	}
	if len(texasCfg.DbInstances) != 1 {
		t.Error(debugVariable(texasCfg.DbInstances))
	}
	if texasCfg.NacosConf.IPAddr != "192.168.50.250" {
		t.Error(debugVariable(texasCfg.NacosConf.IPAddr))
	}
	if texasCfg.NacosConf.GroupName != "poker_gameconf" {
		t.Error(debugVariable(texasCfg.NacosConf.GroupName))
	}
	if texasCfg.SelfBusId != "1.1.80.2" {
		t.Error(debugVariable(texasCfg.SelfBusId))
	}
	if texasCfg.BusMQAddr != "amqp://guest:guest@192.168.50.250:5672/" {
		t.Error(debugVariable(texasCfg.BusMQAddr))
	}
	if texasCfg.RegisterAddr != "192.168.50.250:2182" {
		t.Error(debugVariable(texasCfg.RegisterAddr))
	}
	if texasCfg.GameDataDir != "../common/gamedata/data" {
		t.Error(debugVariable(texasCfg.GameDataDir))
	}

}
