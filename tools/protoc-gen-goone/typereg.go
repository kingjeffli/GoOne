package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

type GoType struct {
	ProtoFullName string // without leading dot: "pkg.Msg" or "pkg.Outer.Inner"
	ImportPath    string // go import path (from go_package, before ';')
	PkgName       string // go package name (from go_package, after ';' or base)
	TypeName      string // Go identifier, nested -> Outer_Inner
}

type TypeRegistry struct {
	byProto map[string]GoType
}

func BuildTypeRegistry(files []*descriptorpb.FileDescriptorProto) (*TypeRegistry, error) {
	r := &TypeRegistry{byProto: map[string]GoType{}}
	for _, fd := range files {
		if fd == nil {
			continue
		}
		pkg := fd.GetPackage()
		goPkgOpt := fd.GetOptions().GetGoPackage()
		importPath, pkgName := splitGoPackage(goPkgOpt, pkg)
		if importPath == "" {
			// For Phase A+, require go_package to resolve cross-package types reliably.
			continue
		}

		for _, m := range fd.GetMessageType() {
			registerMessage(r.byProto, pkg, importPath, pkgName, nil, m)
		}
	}
	return r, nil
}

func (r *TypeRegistry) Resolve(fullType string) (GoType, bool) {
	fullType = strings.TrimSpace(fullType)
	fullType = strings.TrimPrefix(fullType, ".")
	gt, ok := r.byProto[fullType]
	return gt, ok
}

func registerMessage(dst map[string]GoType, protoPkg, importPath, pkgName string, parents []string, m *descriptorpb.DescriptorProto) {
	if m == nil {
		return
	}
	name := m.GetName()
	if name == "" {
		return
	}

	protoFull := protoPkg + "." + name
	if len(parents) > 0 {
		protoFull = protoPkg + "." + strings.Join(append(parents, name), ".")
	}

	goTypeName := name
	if len(parents) > 0 {
		goTypeName = strings.Join(append(parents, name), "_")
	}

	// only register if not already present (first wins)
	if _, exists := dst[protoFull]; !exists {
		dst[protoFull] = GoType{
			ProtoFullName: protoFull,
			ImportPath:    importPath,
			PkgName:       pkgName,
			TypeName:      goTypeName,
		}
	}

	// nested messages
	nextParents := append(parents, name)
	for _, n := range m.GetNestedType() {
		// skip map_entry generated messages
		if n.GetOptions().GetMapEntry() {
			continue
		}
		registerMessage(dst, protoPkg, importPath, pkgName, nextParents, n)
	}
}

func splitGoPackage(goPackageOpt string, protoPkg string) (importPath string, pkgName string) {
	goPackageOpt = strings.TrimSpace(goPackageOpt)
	if goPackageOpt != "" {
		if i := strings.LastIndex(goPackageOpt, ";"); i >= 0 {
			importPath = strings.TrimSpace(goPackageOpt[:i])
			pkgName = strings.TrimSpace(goPackageOpt[i+1:])
			if pkgName == "" {
				pkgName = filepath.Base(importPath)
			}
			return importPath, sanitizeGoIdent(pkgName)
		}
		importPath = goPackageOpt
		pkgName = filepath.Base(importPath)
		if pkgName == "" || pkgName == "." || pkgName == "/" {
			pkgName = ""
		}
		return importPath, sanitizeGoIdent(pkgName)
	}
	// fallback to proto pkg last segment
	if protoPkg != "" {
		parts := strings.Split(protoPkg, ".")
		return "", sanitizeGoIdent(parts[len(parts)-1])
	}
	return "", ""
}

type importSpec struct {
	Path  string
	Alias string
}

// importBuilder assigns unique aliases for imports within a generated file.
type importBuilder struct {
	usedAliases map[string]bool
	byPath      map[string]string
}

func newImportBuilder() *importBuilder {
	return &importBuilder{
		usedAliases: map[string]bool{},
		byPath:      map[string]string{},
	}
}

func (b *importBuilder) add(path, preferredAlias string) (alias string) {
	if path == "" {
		return ""
	}
	if a, ok := b.byPath[path]; ok {
		return a
	}
	alias = strings.TrimSpace(preferredAlias)
	if alias == "" {
		alias = filepath.Base(path)
	}
	alias = sanitizeGoIdent(alias)
	if alias == "" {
		alias = "p"
	}
	if !b.usedAliases[alias] {
		b.usedAliases[alias] = true
		b.byPath[path] = alias
		return alias
	}
	// collision: append suffix
	for i := 2; ; i++ {
		a2 := fmt.Sprintf("%s%d", alias, i)
		if !b.usedAliases[a2] {
			b.usedAliases[a2] = true
			b.byPath[path] = a2
			return a2
		}
	}
}


