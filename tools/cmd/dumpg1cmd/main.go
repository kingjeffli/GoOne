package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	g1_protocol "github.com/Iori372552686/game_protocol/protocol"
)

// Usage examples:
//   go run ./tools/cmd/dumpg1cmd -prefix CMD_ROOM_CENTER_INNER_
//   go run ./tools/cmd/dumpg1cmd -exact CMD_MAIN_LOGIN_REQ
func main() {
	prefix := flag.String("prefix", "", "filter by name prefix (e.g. CMD_ROOM_CENTER_INNER_)")
	exact := flag.String("exact", "", "print only exact name (e.g. CMD_MAIN_LOGIN_REQ)")
	hex := flag.Bool("hex", true, "print value in hex")
	flag.Parse()

	type item struct {
		name string
		val  int32
	}
	items := make([]item, 0, len(g1_protocol.CMD_name))
	for v, n := range g1_protocol.CMD_name {
		if *exact != "" && n != *exact {
			continue
		}
		if *prefix != "" && !strings.HasPrefix(n, *prefix) {
			continue
		}
		items = append(items, item{name: n, val: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].val != items[j].val {
			return items[i].val < items[j].val
		}
		return items[i].name < items[j].name
	})

	for _, it := range items {
		if *hex {
			fmt.Printf("%s\t0x%X\t(%d)\n", it.name, uint32(it.val), it.val)
		} else {
			fmt.Printf("%s\t%d\n", it.name, it.val)
		}
	}
}


