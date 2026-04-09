package test

import (
	"testing"

	"path/filepath"

	"github.com/Iori372552686/GoOne/tools/cfgtool/domain"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/base"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/manager"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/parser"
	"github.com/Iori372552686/GoOne/tools/cfgtool/service"
)

func TestConfig(t *testing.T) {
	domain.XlsxPath = "../xls"
	domain.JsonPath = "../gen/json"
	domain.ProtoPath = "../gen/proto"
	domain.CodePath = "../gen/code"
	domain.LuaPath = "../gen/lua"
	domain.Module = "github.com/Iori372552686/GoOne"
	domain.PbPath = "github.com/Iori372552686/game_protocol/protocol"
	domain.PkgName = filepath.Base(domain.PbPath)

	// 加载所有配置
	files, err := base.Glob(domain.XlsxPath, ".*\\.xlsx", true)
	if err != nil {
		panic(err)
	}
	// 解析所有文件
	if err := parser.ParseFiles(files...); err != nil {
		panic(err)
	}
	// 生成proto文件数据
	if err := service.GenProto(); err != nil {
		panic(err)
	}
	if err := service.SaveProto(); err != nil {
		panic(err)
	}
	// 解析proto文件
	if err := manager.ParseProto(); err != nil {
		panic(err)
	}
	if err := service.GenData(); err != nil {
		panic(err)
	}
	if err := service.GenCode(); err != nil {
		panic(err)
	}
}
