package service

import (
	"bytes"
	"path"
	"strings"

	"github.com/Iori372552686/GoOne/tools/cfgtool/domain"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/base"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/errs"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/logx"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/manager"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/templ"
	"github.com/iancoleman/strcase"
)

// CppConfigInfo 传给 C++ 模板的数据
type CppConfigInfo struct {
	FileName      string       // proto 文件名（不含扩展名）
	DataName      string       // 配置名去掉 Config 后缀，如 "Item"
	CppNamespace  string       // C++ 命名空间，如 "g1::protocol"
	ProtoFullType string       // proto message 全限定名，如 "g1::protocol::ItemConfig"
	Config        *base.Config // 配置元数据
}

func toCppNamespace(protoPkg string) string {
	if protoPkg == "" {
		return "g1::protocol"
	}
	return strings.ReplaceAll(protoPkg, ".", "::")
}

// GenCpp 生成 C++17 版本的配置加载与查询便捷代码（可选）
func GenCpp() error {
	if len(domain.PbPath) <= 0 || len(domain.CppPath) <= 0 {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	ns := toCppNamespace(domain.ProtoPkgName)

	for _, st := range manager.GetConfigMap() {
		buf.Reset()
		dataName := strings.TrimSuffix(st.Name, "Config")
		dirName := strcase.ToSnake(st.Name)
		item := &CppConfigInfo{
			FileName:      st.FileName,
			DataName:      dataName,
			CppNamespace:  ns,
			ProtoFullType: ns + "::" + st.Name,
			Config:        st,
		}
		if err := templ.CppTpl.Execute(buf, item); err != nil {
			return errs.Wrap(err, st.FileName, st.Sheet, "", 0, "生成错误", "渲染C++模板失败")
		}
		if err := base.Save(domain.CppPath, path.Join(dirName, dataName+"Data.gen.hpp"), buf.Bytes()); err != nil {
			return errs.Wrap(err, st.FileName, st.Sheet, "", 0, "保存错误", "保存C++代码失败")
		}
	}

	logx.Successf("C++代码生成完成")
	return nil
}
