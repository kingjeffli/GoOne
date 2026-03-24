package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func main() {
	var (
		module    = flag.String("module", "github.com/Iori372552686/GoOne", "Go module path (used for output layout)")
		outDir    = flag.String("out", ".", "output directory (typically repo root)")
		protoRoot = flag.String("proto_root", "api/proto", "proto root directory")
		protoc    = flag.String("protoc", "", "path to protoc (optional). If empty, tries PATH then repo-vendored (linux only).")
	)
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		die(err)
	}

	absOut := filepath.Join(repoRoot, *outDir)
	absProtoRoot := filepath.Join(repoRoot, *protoRoot)

	// Ensure cmd.proto exists (new checkouts shouldn't fail due to missing generated IDL).
	if err := ensureCmdProto(repoRoot, absProtoRoot); err != nil {
		die(err)
	}

	protocPath, err := findProtoc(repoRoot, *protoc)
	if err != nil {
		die(err)
	}

	binDir := filepath.Join(repoRoot, ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		die(err)
	}

	protocGenGo := filepath.Join(binDir, exeName("protoc-gen-go"))
	protocGenGoone := filepath.Join(binDir, exeName("protoc-gen-goone"))

	// build plugins (deterministic, pinned to repo go.mod)
	if err := goBuild(repoRoot, protocGenGoone, "./tools/protoc-gen-goone"); err != nil {
		die(err)
	}
	if err := goBuild(repoRoot, protocGenGo, "google.golang.org/protobuf/cmd/protoc-gen-go"); err != nil {
		die(err)
	}

	// proto inputs: only api/proto/goone and api/proto/game
	inputs, err := collectProtoInputs(absProtoRoot)
	if err != nil {
		die(err)
	}
	if len(inputs) == 0 {
		die(errors.New("no proto files found under api/proto/{goone,game}"))
	}

	// include paths:
	// - api/proto
	// - google well-known types from vendored protoc include dirs (if present)
	includePaths := []string{absProtoRoot}
	includePaths = append(includePaths, existingGoogleIncludes(repoRoot)...)

	args := []string{}
	for _, inc := range includePaths {
		args = append(args, "-I", inc)
	}

	args = append(args,
		fmt.Sprintf("--plugin=protoc-gen-go=%s", protocGenGo),
		fmt.Sprintf("--plugin=protoc-gen-goone=%s", protocGenGoone),
		fmt.Sprintf("--go_out=%s", absOut),
		fmt.Sprintf("--go_opt=module=%s", *module),
		"--go_opt=paths=import",
		fmt.Sprintf("--goone_out=%s", absOut),
		fmt.Sprintf("--goone_opt=module=%s", *module),
		"--goone_opt=paths=import",
	)

	args = append(args, inputs...)

	cmd := exec.Command(protocPath, args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[genproto] protoc=%s\n", protocPath)
	fmt.Printf("[genproto] module=%s\n", *module)
	fmt.Printf("[genproto] inputs=%d\n", len(inputs))
	if err := cmd.Run(); err != nil {
		die(fmt.Errorf("protoc failed: %w", err))
	}
	fmt.Println("[genproto] done")
}

func die(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "[genproto] error:", err)
	os.Exit(1)
}

func ensureCmdProto(repoRoot, absProtoRoot string) error {
	cmdProto := filepath.Join(absProtoRoot, "goone", "cmd", "v1", "cmd.proto")
	_, statErr := os.Stat(cmdProto)
	if statErr == nil && os.Getenv("GEN_CMD_PROTO") != "1" {
		return nil
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) && os.Getenv("GEN_CMD_PROTO") != "1" {
		return fmt.Errorf("stat cmd.proto: %w", statErr)
	}

	// Generate via go run so this works cross-platform (no shell script dependency).
	fmt.Printf("[genproto] ensure cmd.proto -> %s\n", cmdProto)
	cmd := exec.Command("go", "run", "./tools/cmd/gencmdproto")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generate cmd.proto (gencmdproto): %w", err)
	}
	return nil
}

func exeName(name string) string {
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return name + ".exe"
	}
	return name
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return "", errors.New("cannot locate repo root (go.mod not found)")
}

func goBuild(repoRoot, out, pkg string) error {
	args := []string{"build", "-o", out, pkg}
	cmd := exec.Command("go", args...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findProtoc(repoRoot string, override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", fmt.Errorf("protoc not found at %q: %w", override, err)
		}
		return override, nil
	}
	if p, err := exec.LookPath(exeName("protoc")); err == nil {
		return p, nil
	}
	// fallback: vendored linux protoc (works in WSL/Linux)
	if runtime.GOOS == "linux" {
		p := filepath.Join(repoRoot, "lib", "contrib", "protoc", "protoc-33.2-linux-x86_64", "bin", "protoc")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		p2 := filepath.Join(repoRoot, "lib", "util", "deps", "protoc", "protoc-33.2-linux-x86_64", "bin", "protoc")
		if _, err := os.Stat(p2); err == nil {
			return p2, nil
		}
	}
	return "", errors.New("protoc not found on PATH. On Windows, install protoc or run from WSL; on Linux, ensure protoc exists or use vendored one.")
}

func existingGoogleIncludes(repoRoot string) []string {
	candidates := []string{
		filepath.Join(repoRoot, "lib", "contrib", "protoc", "protoc-30.1-win64", "include"),
		filepath.Join(repoRoot, "lib", "contrib", "protoc", "protoc-33.2-linux-x86_64", "include"),
		filepath.Join(repoRoot, "lib", "util", "deps", "protoc", "protoc-3.11.4-win64", "include"),
		filepath.Join(repoRoot, "lib", "util", "deps", "protoc", "protoc-33.2-linux-x86_64", "include"),
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] {
			continue
		}
		seen[c] = true
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			out = append(out, c)
		}
	}
	return out
}

func collectProtoInputs(absProtoRoot string) ([]string, error) {
	roots := []string{
		filepath.Join(absProtoRoot, "goone"),
		filepath.Join(absProtoRoot, "game"),
	}
	var protos []string
	for _, r := range roots {
		_ = filepath.WalkDir(r, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".proto") {
				// protoc wants paths relative to proto_root for clean import mapping
				rel, err := filepath.Rel(absProtoRoot, path)
				if err != nil {
					return err
				}
				rel = filepath.ToSlash(rel)
				protos = append(protos, rel)
			}
			return nil
		})
	}
	sort.Strings(protos)
	return protos, nil
}
