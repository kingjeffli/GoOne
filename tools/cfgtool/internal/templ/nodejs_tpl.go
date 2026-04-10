package templ

import (
	"strings"
	"text/template"

	"github.com/Iori372552686/GoOne/tools/cfgtool/domain"
	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/base"
)

// proto 基础类型 -> TypeScript 类型
func protoToTs(name string) string {
	switch name {
	case "int32", "int64", "uint32", "uint64":
		return "number"
	case "float", "double":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	default:
		return "any"
	}
}

// 单个字段的TS类型（用于Index键类型）
func tsFieldType(f *base.Field) string {
	return tsType(f.Type)
}

// Field.Type -> 完整TS类型字符串
func tsType(t *base.Type) string {
	depth := t.ArrayDepth
	if depth == 0 && t.ValueOf == domain.ValueOfList {
		depth = 1
	}

	var baseType string
	switch t.TypeOf {
	case domain.TypeOfEnum:
		baseType = "number"
	case domain.TypeOfStruct, domain.TypeOfConfig:
		baseType = "I" + t.Name
	default: // TypeOfBase
		baseType = protoToTs(t.Name)
	}
	return baseType + strings.Repeat("[]", depth)
}

// Index -> TS 函数参数签名，如 "Id: number, Level: number"
func tsArg(idx *base.Index) string {
	parts := []string{}
	for _, f := range idx.List {
		parts = append(parts, f.Name+": "+tsFieldType(f))
	}
	return strings.Join(parts, ", ")
}

// Index -> Map 键类型（单字段用原始类型，多字段用 string，空 List 用 string）
func tsKeyType(idx *base.Index) string {
	if len(idx.List) == 0 {
		return "string"
	}
	if len(idx.List) == 1 {
		return tsFieldType(idx.List[0])
	}
	return "string"
}

// Index -> 取键表达式，ref 为对象引用（如 "item"），为空则直接用参数名
func tsKey(idx *base.Index, ref string) string {
	if len(idx.List) == 0 {
		return `""`
	}
	if len(idx.List) == 1 {
		f := idx.List[0]
		if ref != "" {
			return ref + "." + f.Name
		}
		return f.Name
	}
	// 复合键：用字符串拼接（避免模板中出现反引号）
	parts := []string{}
	for _, f := range idx.List {
		if ref != "" {
			parts = append(parts, "String("+ref+"."+f.Name+")")
		} else {
			parts = append(parts, "String("+f.Name+")")
		}
	}
	return strings.Join(parts, ` + "_" + `)
}

const nodejsTpl = `// ============================================================================
// 本代码由xlsx工具自动生成，请勿手动修改
// Config: {{.Config.Name}}
// ============================================================================
{{range $st := .StructDefs}}
/** {{$st.Name}} 子结构接口 */
export interface I{{$st.Name}} {
{{- range $field := $st.FieldList}}
    /** {{$field.Desc}} */
    readonly {{$field.Name}}: {{tstype $field.Type}};
{{- end}}
}
{{end}}
/** {{.Config.Name}} 数据接口 */
export interface I{{.Config.Name}} {
{{- range $field := .Config.FieldList}}
    /** {{$field.Desc}} */
    readonly {{$field.Name}}: {{tstype $field.Type}};
{{- end}}
}

type Item = I{{.Config.Name}};
type Predicate = (item: Item) => boolean;

// ---------------------------------------------------------------------------
//  {{.DataName}} 配置管理器（单例，不可变快照）
// ---------------------------------------------------------------------------
class {{.DataName}}DataMgr {
    private _list: ReadonlyArray<Item> = [];
{{- range $index := .Config.IndexList}}
{{- if eq $index.Type.ValueOf 3}}
    private _map{{$index.Name}} = new Map<{{tskeytype $index}}, Item>();
{{- else if eq $index.Type.ValueOf 4}}
    private _group{{$index.Name}} = new Map<string, ReadonlyArray<Item>>();
{{- end}}
{{- end}}

    // =======================================================================
    //  加载
    // =======================================================================

    /** 从 JSON 数组加载配置数据，构建全部索引 */
    Parse(data: Item[]): void {
        this._list = Object.freeze(data);
{{- range $index := .Config.IndexList}}
{{- if eq $index.Type.ValueOf 3}}
        const m{{$index.Name}} = new Map<{{tskeytype $index}}, Item>();
        for (const item of data) {
            m{{$index.Name}}.set({{tskey $index "item"}}, item);
        }
        this._map{{$index.Name}} = m{{$index.Name}};
{{- else if eq $index.Type.ValueOf 4}}
        const g{{$index.Name}} = new Map<string, Item[]>();
        for (const item of data) {
            const gk = String({{tskey $index "item"}});
            let arr = g{{$index.Name}}.get(gk);
            if (!arr) { arr = []; g{{$index.Name}}.set(gk, arr); }
            arr.push(item);
        }
        this._group{{$index.Name}} = g{{$index.Name}};
{{- end}}
{{- end}}
    }

    // =======================================================================
    //  基础查询
    // =======================================================================

    /** 获取第一条记录 */
    GetHead(): Item | undefined {
        return this._list[0];
    }

    /** 获取全部记录（只读视图，零拷贝） */
    GetAll(): ReadonlyArray<Item> {
        return this._list;
    }

    /** 记录总数 */
    Count(): number {
        return this._list.length;
    }

    // =======================================================================
    //  遍历
    // =======================================================================

    /** 遍历所有记录，fn 返回 false 时提前终止 */
    Range(fn: (item: Item) => boolean): void {
        for (let i = 0, n = this._list.length; i < n; i++) {
            if (!fn(this._list[i])) return;
        }
    }

    // =======================================================================
    //  条件查询
    // =======================================================================

    /** 返回第一个满足条件的记录 */
    Find(fn: Predicate): Item | undefined {
        for (let i = 0, n = this._list.length; i < n; i++) {
            if (fn(this._list[i])) return this._list[i];
        }
        return undefined;
    }

    /** 返回所有满足条件的记录 */
    Filter(fn: Predicate): Item[] {
        const out: Item[] = [];
        for (let i = 0, n = this._list.length; i < n; i++) {
            if (fn(this._list[i])) out.push(this._list[i]);
        }
        return out;
    }

    // =======================================================================
    //  索引查询
    // =======================================================================
{{- range $index := .Config.IndexList}}
{{- if eq $index.Type.ValueOf 3}}

    /** 按 {{$index.Name}} 精确查找 */
    GetBy{{$index.Name}}({{tsarg $index}}): Item | undefined {
        return this._map{{$index.Name}}.get({{tskey $index ""}});
    }
{{- else if eq $index.Type.ValueOf 4}}

    /** 按 {{$index.Name}} 分组查找 */
    GroupBy{{$index.Name}}({{tsarg $index}}): ReadonlyArray<Item> {
        return this._group{{$index.Name}}.get(String({{tskey $index ""}})) ?? [];
    }
{{- end}}
{{- end}}
}

/** 单例导出 */
export const {{.VarName}} = new {{.DataName}}DataMgr();
`

var NodeJsTpl = template.Must(
	template.New("NodeJsTpl").Funcs(template.FuncMap{
		"tstype":    tsType,
		"tsarg":     tsArg,
		"tskeytype": tsKeyType,
		"tskey":     tsKey,
	}).Parse(nodejsTpl),
)
