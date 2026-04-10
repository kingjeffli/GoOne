package templ

import (
	"text/template"

	"github.com/Iori372552686/GoOne/tools/cfgtool/internal/base"
)

const cppTpl = `
// ============================================================================
// 本代码由xlsx工具自动生成，请勿手动修改
// Config: {{.Config.Name}}
// ============================================================================

#pragma once

#include <cstdint>
#include <string>
#include <vector>
#include <unordered_map>
#include <functional>
#include <memory>
#include <shared_mutex>
#include <optional>
#include <sstream>

#include <google/protobuf/util/json_util.h>
#include "{{.FileName}}.pb.h"

namespace {{.CppNamespace}} {

// ------------------------------------------------------------------------
// {{.DataName}}Data — 配置加载 & 便捷查询（线程安全，C++17）
// ------------------------------------------------------------------------
class {{.DataName}}Data final {
public:
    using Item    = {{.ProtoFullType}};
    using ItemAry = {{.ProtoFullType}}Ary;
    using Ptr     = const Item*;
    using PredFn  = std::function<bool(Ptr)>;

    {{.DataName}}Data() = default;
    ~{{.DataName}}Data() = default;

    // 禁止拷贝
    {{.DataName}}Data(const {{.DataName}}Data&) = delete;
    {{.DataName}}Data& operator=(const {{.DataName}}Data&) = delete;

    // ====================================================================
    //  加载
    // ====================================================================

    /** 从 JSON 字符串加载配置，线程安全 */
    static bool Load(const std::string& json, std::string* err_out = nullptr) {
        auto storage = std::make_shared<ItemAry>();
        google::protobuf::util::JsonParseOptions opts;
        opts.ignore_unknown_fields = true;
        auto status = google::protobuf::util::JsonStringToMessage(json, storage.get());
        if (!status.ok()) {
            if (err_out) *err_out = std::string(status.message());
            return false;
        }

        auto d = std::make_shared<Inner>();
        d->storage = std::move(storage);
        d->list.reserve(d->storage->ary_size());
        for (const auto& item : d->storage->ary()) {
            d->list.push_back(&item);
{{- range $index := .Config.IndexList}}
{{- if eq $index.Type.ValueOf 3 }}
            d->map_{{$index.Name}}[{{cppMapKey $index "item"}}] = &item;
{{- else if eq $index.Type.ValueOf 4 }}
            d->group_{{$index.Name}}[{{cppGroupKey $index "item"}}].push_back(&item);
{{- end}}
{{- end}}
        }

        std::unique_lock lock(Mu());
        Data() = std::move(d);
        return true;
    }

    // ====================================================================
    //  基础查询
    // ====================================================================

    /** 获取第一条记录 */
    static Ptr GetHead() {
        auto d = Snapshot();
        return (d && !d->list.empty()) ? d->list.front() : nullptr;
    }

    /** 获取全部记录 */
    static std::vector<Ptr> GetAll() {
        auto d = Snapshot();
        return d ? d->list : std::vector<Ptr>{};
    }

    /** 记录总数 */
    static size_t Count() {
        auto d = Snapshot();
        return d ? d->list.size() : 0;
    }

    // ====================================================================
    //  遍历
    // ====================================================================

    /** 遍历所有记录，fn 返回 false 时提前终止 */
    static void Range(const PredFn& fn) {
        auto d = Snapshot();
        if (!d) return;
        for (auto* item : d->list) {
            if (!fn(item)) return;
        }
    }

    // ====================================================================
    //  条件查询
    // ====================================================================

    /** 返回第一个满足条件的记录 */
    static Ptr Find(const PredFn& fn) {
        auto d = Snapshot();
        if (!d) return nullptr;
        for (auto* item : d->list) {
            if (fn(item)) return item;
        }
        return nullptr;
    }

    /** 返回所有满足条件的记录 */
    static std::vector<Ptr> Filter(const PredFn& fn) {
        std::vector<Ptr> out;
        auto d = Snapshot();
        if (!d) return out;
        for (auto* item : d->list) {
            if (fn(item)) out.push_back(item);
        }
        return out;
    }

    // ====================================================================
    //  索引查询
    // ====================================================================
{{- range $index := .Config.IndexList}}
{{- if eq $index.Type.ValueOf 3 }}

    /** 按 {{$index.Name}} 精确查找 */
    static Ptr GetBy{{$index.Name}}({{$index.CppArg ", "}}) {
        auto d = Snapshot();
        if (!d) return nullptr;
        auto it = d->map_{{$index.Name}}.find({{cppMapKey $index ""}});
        return it != d->map_{{$index.Name}}.end() ? it->second : nullptr;
    }
{{- else if eq $index.Type.ValueOf 4 }}

    /** 按 {{$index.Name}} 分组查找 */
    static std::vector<Ptr> GroupBy{{$index.Name}}({{$index.CppArg ", "}}) {
        auto d = Snapshot();
        if (!d) return {};
        auto it = d->group_{{$index.Name}}.find({{cppGroupKey $index ""}});
        return it != d->group_{{$index.Name}}.end() ? it->second : std::vector<Ptr>{};
    }
{{- end}}
{{- end}}

private:
    struct Inner {
        std::shared_ptr<ItemAry>    storage;
        std::vector<Ptr>            list;
{{- range $index := .Config.IndexList}}
{{- if eq $index.Type.ValueOf 3 }}
        std::unordered_map<{{$index.CppKeyType}}, Ptr> map_{{$index.Name}};
{{- else if eq $index.Type.ValueOf 4 }}
        std::unordered_map<std::string, std::vector<Ptr>> group_{{$index.Name}};
{{- end}}
{{- end}}
    };

    static std::shared_ptr<Inner>& Data() {
        static std::shared_ptr<Inner> inst;
        return inst;
    }
    static std::shared_mutex& Mu() {
        static std::shared_mutex mu;
        return mu;
    }
    /** 读快照：共享锁下拷贝 shared_ptr（引用计数 +1，零拷贝数据） */
    static std::shared_ptr<Inner> Snapshot() {
        std::shared_lock lock(Mu());
        return Data();
    }
};

} // namespace {{.CppNamespace}}
`

// cppMapKey 生成 map 索引的键表达式
// 单字段：直接用原始类型值；多字段：ostringstream 拼接为 string
func cppMapKey(idx *base.Index, ref string) string {
	if len(idx.List) == 0 {
		return `""`
	}
	return idx.CppKeyExpr(ref)
}

// cppGroupKey group 索引始终拼接为 string
func cppGroupKey(idx *base.Index, ref string) string {
	if len(idx.List) == 0 {
		return `""`
	}
	if len(idx.List) == 1 {
		// 单字段也用 std::to_string 转为 string key
		f := idx.List[0]
		if f.Type.Name == "string" {
			if ref != "" {
				return ref + "." + toLower(f.Name) + "()"
			}
			return f.Name
		}
		if ref != "" {
			return "std::to_string(" + ref + "." + toLower(f.Name) + "())"
		}
		return "std::to_string(" + f.Name + ")"
	}
	return idx.CppKeyExpr(ref)
}

func toLower(s string) string {
	if s == "" {
		return s
	}
	// simple: lowercase first char, keep rest (proto field convention)
	r := []rune(s)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] = r[0] + 32
	}
	return string(r)
}

var CppTpl = template.Must(
	template.New("CppTpl").Funcs(template.FuncMap{
		"cppMapKey":   cppMapKey,
		"cppGroupKey": cppGroupKey,
	}).Parse(cppTpl),
)
