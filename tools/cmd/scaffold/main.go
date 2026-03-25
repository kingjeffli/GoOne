// Package main implements a lightweight scaffold tool for creating new GoOne server skeletons.
//
// Usage:
//
//	go run tools/cmd/scaffold -name mysvr
//	go run tools/cmd/scaffold -name mysvr -module github.com/Iori372552686/GoOne -out src/
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

func main() {
	var (
		name   = flag.String("name", "", "server name (required, e.g. mysvr)")
		module = flag.String("module", "github.com/Iori372552686/GoOne", "Go module path")
		out    = flag.String("out", "src", "output parent directory (server dir is created inside)")
	)
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: -name is required")
		flag.Usage()
		os.Exit(1)
	}

	svrName := strings.ToLower(strings.TrimSpace(*name))
	if !strings.HasSuffix(svrName, "svr") {
		svrName += "svr"
	}

	data := templateData{
		Name:       svrName,
		StructName: toPascalCase(svrName),
		ImplName:   toPascalCase(svrName) + "Impl",
		Module:     *module,
	}

	svrDir := filepath.Join(*out, svrName)
	if err := generateAll(svrDir, data); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s/\n", svrDir)
	fmt.Println("Next steps:")
	fmt.Printf("  1. Add config in common/gconf/config.go:  type %s struct { SelfBusId, LogDir, LogLevel string }\n", data.StructName)
	fmt.Printf("  2. Add config YAML section for %s\n", svrName)
	fmt.Printf("  3. Define proto service in api/proto/game/%s/v1/\n", strings.TrimSuffix(svrName, "svr"))
	fmt.Printf("  4. Run: go build ./src/%s/\n", svrName)
}

type templateData struct {
	Name       string // e.g. "mysvr"
	StructName string // e.g. "Mysvr"
	ImplName   string // e.g. "MysvrImpl"
	Module     string // e.g. "github.com/Iori372552686/GoOne"
}

func generateAll(svrDir string, data templateData) error {
	files := []struct {
		relPath  string
		template string
	}{
		{"main.go", tplMainGo},
		{"app.go", tplAppGo},
		{filepath.Join("globals", "globals.go"), tplGlobalsGo},
		{filepath.Join("cmd_handler", "register.go"), tplRegisterGo},
	}

	for _, f := range files {
		outPath := filepath.Join(svrDir, f.relPath)
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("file already exists: %s (refusing to overwrite)", outPath)
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}

		tmpl, err := template.New(f.relPath).Parse(f.template)
		if err != nil {
			return fmt.Errorf("template parse %s: %w", f.relPath, err)
		}

		w, err := os.Create(outPath)
		if err != nil {
			return err
		}
		if err := tmpl.Execute(w, data); err != nil {
			w.Close()
			return fmt.Errorf("template exec %s: %w", f.relPath, err)
		}
		w.Close()
	}
	return nil
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

var tplMainGo = `package main

import (
	"flag"

	"{{.Module}}/lib/api/logger"
	"{{.Module}}/lib/service/application"
)

func main() {
	flag.Parse()
	defer logger.Flush()

	application.Init(&{{.ImplName}}{})
	application.Run()
}
`

var tplAppGo = `package main

import (
	"runtime"

	"{{.Module}}/common/gconf"
	"{{.Module}}/lib/api/logger"
	"{{.Module}}/lib/api/sharedstruct"
	"{{.Module}}/lib/service/router"
	"{{.Module}}/lib/util/marshal"
	"{{.Module}}/module/misc"
	"{{.Module}}/src/{{.Name}}/cmd_handler"
	"{{.Module}}/src/{{.Name}}/globals"
)

func onRecvSSPacket(packet *sharedstruct.SSPacket) {
	globals.TransMgr.ProcessSSPacket(packet)
	packet = nil
}

type {{.ImplName}} struct{}

func (a *{{.ImplName}}) OnInit() error {
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)

	if err := a.OnReload(); err != nil {
		logger.Errorf("Failed to load config | %v", err)
		return err
	}

	// TODO: update gconf reference after adding config struct.
	// if _, err := logger.InitLogger(gconf.{{.StructName}}SvrCfg.LogDir, gconf.{{.StructName}}SvrCfg.LogLevel, "{{.Name}}"); err != nil {
	// 	return err
	// }

	// TODO: init router after adding config.
	// err := router.InitAndRun(gconf.{{.StructName}}SvrCfg.SelfBusId,
	// 	onRecvSSPacket,
	// 	gconf.{{.StructName}}SvrCfg.BusMQAddr,
	// 	misc.ServerRouteRules,
	// 	gconf.{{.StructName}}SvrCfg.RegisterAddr,
	// )
	// if err != nil {
	// 	return err
	// }

	cmd_handler.RegCmd()
	globals.TransMgr.InitAndRun(misc.MaxTransNumber, false, 0)

	logger.Infof("{{.Name}} init success")
	return nil
}

func (a *{{.ImplName}}) OnReload() error {
	// TODO: update to load your config struct.
	// err := marshal.LoadConfFile(*gconf.SvrConfFile, &gconf.{{.StructName}}SvrCfg)
	// if err != nil {
	// 	logger.Errorf("failed to load svr config | %%s", err)
	// 	return err
	// }
	return nil
}

func (a *{{.ImplName}}) OnProc() bool {
	return true
}

func (a *{{.ImplName}}) OnTick(lastMs, nowMs int64) {
}

func (a *{{.ImplName}}) OnExit() {
	logger.Flush()
	logger.Infof("service exit, right now !")
	logger.Infof("================== {{.ImplName}} Stop =========================")
}

// Suppress unused import warnings during scaffold phase.
var (
	_ = gconf.SvrConfFile
	_ = router.InitAndRun
	_ = marshal.LoadConfFile
	_ = (*sharedstruct.SSPacket)(nil)
)
`

var tplGlobalsGo = `package globals

import (
	"{{.Module}}/lib/service/transaction"
)

var (
	TransMgr = transaction.NewTransactionMgr()
	// TODO: add domain-specific managers here, e.g.:
	// RedisMgr = redis.NewRedisMgr()
)
`

var tplRegisterGo = `package cmd_handler

import (
	"{{.Module}}/lib/api/logger"
	// "{{.Module}}/src/{{.Name}}/globals"
	// g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

// RegCmd registers all command handlers for {{.Name}}.
func RegCmd() {
	logger.Infof("register transaction commands")
	// If this service is IDL-first, generated ssrpc registration will usually be
	// wired from app.go instead, and this file can stay empty or be removed later.
	// TODO: register your cmd handlers, e.g.:
	// globals.TransMgr.RegisterCmd(g1_protocol.CMD_XXX_REQ, YourHandler)
}
`
