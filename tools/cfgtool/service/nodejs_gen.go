package service

import (
	"bytes"
	"path"
	"strings"
	"unicode"

	"github.com/Iori372552686/GoOne/tools/cfgtool/domain"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/base"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/errs"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/logx"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/manager"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/templ"
	"github.com/iancoleman/strcase"
)

// NodeJsConfigInfo 传给 Node.js/TypeScript 模板的数据
type NodeJsConfigInfo struct {
	DataName   string         // 配置名去掉 Config 后缀，如 "Item"
	VarName    string         // 导出变量名（小驼峰），如 "item"
	Config     *base.Config   // 配置元数据
	StructDefs []*base.Struct // 本配置引用到的结构体定义
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func collectStructDefs(fields []*base.Field, seen map[string]bool, defs *[]*base.Struct) {
	for _, field := range fields {
		if field.Type.TypeOf != domain.TypeOfStruct || seen[field.Type.Name] {
			continue
		}
		st := manager.GetStruct(field.Type.Name)
		if st == nil {
			continue
		}
		seen[field.Type.Name] = true
		*defs = append(*defs, st)
		collectStructDefs(st.FieldList, seen, defs)
	}
}

// GenNodeJs 生成 Node.js/TypeScript 版本的配置加载与查询代码（可选）
func GenNodeJs() error {
	if len(domain.NodeJsPath) <= 0 {
		return nil
	}

	buf := bytes.NewBuffer(nil)

	for _, st := range manager.GetConfigMap() {
		buf.Reset()
		dataName := strings.TrimSuffix(st.Name, "Config")
		dirName := strcase.ToSnake(st.Name)

		structDefs := []*base.Struct{}
		seen := map[string]bool{}
		collectStructDefs(st.FieldList, seen, &structDefs)

		item := &NodeJsConfigInfo{
			DataName:   dataName,
			VarName:    toLowerFirst(dataName),
			Config:     st,
			StructDefs: structDefs,
		}

		if err := templ.NodeJsTpl.Execute(buf, item); err != nil {
			return errs.Wrap(err, st.FileName, st.Sheet, "", 0, "生成错误", "渲染Node.js模板失败")
		}
		if err := base.Save(domain.NodeJsPath, path.Join(dirName, dataName+"Data.gen.ts"), buf.Bytes()); err != nil {
			return errs.Wrap(err, st.FileName, st.Sheet, "", 0, "保存错误", "保存Node.js代码失败")
		}
	}

	// 生成 index.ts 入口文件，统一导出所有配置
	buf.Reset()
	buf.WriteString("/**\n * 本代码由xlsx工具自动生成，请勿手动修改\n * 配置管理入口\n */\n\n")
	for _, st := range manager.GetConfigMap() {
		dataName := strings.TrimSuffix(st.Name, "Config")
		dirName := strcase.ToSnake(st.Name)
		varName := toLowerFirst(dataName)
		buf.WriteString("export { " + varName + ", type I" + st.Name + " } from './" + dirName + "/" + dataName + "Data.gen';\n")
	}
	if err := base.Save(domain.NodeJsPath, "index.ts", buf.Bytes()); err != nil {
		return errs.Wrap(err, "", "", "", 0, "保存错误", "保存Node.js index.ts失败")
	}

	logx.Successf("Node.js/TypeScript代码生成完成")
	return nil
}
