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

type ConfigInfo struct {
	PbPkg string
	Pkg   string
	*base.Config
}

type IndexInfo struct {
	Pkg       string
	IndexList []int
}

func GenCode() error {
	buf := bytes.NewBuffer(nil)

	if len(domain.PbPath) <= 0 || len(domain.CodePath) <= 0 || len(domain.Module) <= 0 {

		return nil
	}

	// 生成索引
	if err := genIndex(buf); err != nil {
		return errs.Wrap(err, "", "", "", 0, "生成错误", "生成索引失败")
	}
	// 对文件分类
	for _, st := range manager.GetConfigMap() {
		buf.Reset()
		dataName := strings.TrimSuffix(st.Name, "Config")
		name := strcase.ToSnake(st.Name)
		item := &ConfigInfo{
			PbPkg:  domain.PkgName,
			Pkg:    name,
			Config: st,
		}
		if err := templ.CodeTpl.Execute(buf, item); err != nil {
			return errs.Wrap(err, st.FileName, st.Sheet, "", 0, "生成错误", "渲染代码模板失败")
		}
		// 保存代码
		if err := base.SaveGo(path.Join(domain.CodePath, name), dataName+"Data.gen.go", buf.Bytes()); err != nil {
			return errs.Wrap(err, st.FileName, st.Sheet, "", 0, "保存错误", "保存代码失败")
		}
	}

	logx.Successf("Go代码生成完成")
	return nil
}
