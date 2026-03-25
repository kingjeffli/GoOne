package main

import (
	"testing"

	"github.com/Iori372552686/GoOne/lib/net/net_mgr"
)

func TestPickClientPacketRoute_PrefersExactOriginMatch(t *testing.T) {
	routes := []clientPacketRoute{
		{name: "ws", client: &net_mgr.Client{Uid: 1, Ip: 11, Port: 22}},
		{name: "tcp", client: &net_mgr.Client{Uid: 1, Ip: 33, Port: 44}},
	}

	got := pickClientPacketRoute(routes, 33, 44)
	if got == nil || got.name != "tcp" {
		t.Fatalf("expected tcp route, got %#v", got)
	}
}

func TestPickClientPacketRoute_FallsBackToOnlyAvailableRoute(t *testing.T) {
	routes := []clientPacketRoute{
		{name: "ws", client: nil},
		{name: "tcp", client: &net_mgr.Client{Uid: 1, Ip: 33, Port: 44}},
	}

	got := pickClientPacketRoute(routes, 0, 0)
	if got == nil || got.name != "tcp" {
		t.Fatalf("expected tcp route fallback, got %#v", got)
	}
}

func TestPickClientPacketRoute_ReturnsNilWhenNoClient(t *testing.T) {
	routes := []clientPacketRoute{
		{name: "ws", client: nil},
		{name: "tcp", client: nil},
	}

	got := pickClientPacketRoute(routes, 11, 22)
	if got != nil {
		t.Fatalf("expected nil route, got %#v", got)
	}
}
