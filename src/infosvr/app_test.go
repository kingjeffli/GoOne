package main

import (
	"github.com/Iori372552686/GoOne/lib/service/bootstrap"
	"github.com/Iori372552686/GoOne/lib/service/router"
	"github.com/Iori372552686/GoOne/lib/service/transaction"
	"strings"
	"testing"
)

func TestBuildInfoSvrComponentStatuses(t *testing.T) {
	statuses := buildInfoSvrComponentStatuses(2, transaction.TransactionMgrStats{
		ShardCount:         4,
		ActiveTransactions: 3,
		PendingPackets:     5,
		DroppedPackets:     1,
	}, router.AdminSnapshot{
		Initialized:  true,
		SelfBusID:    1025,
		ShuttingDown: false,
	})
	assertInfoComponentStatus(t, statuses, "infosvr.redis", "ready", true)
	assertInfoComponentStatus(t, statuses, "infosvr.transaction_mgr", "ready", true)
	assertInfoComponentStatus(t, statuses, "infosvr.router", "ready", true)
	routerStatus := findInfoComponentStatus(t, statuses, "infosvr.router")
	if !strings.Contains(routerStatus.Message, "bus_id=1025") {
		t.Fatalf("expected router message to include bus id, got %q", routerStatus.Message)
	}
}
func TestBuildInfoSvrComponentStatusesPending(t *testing.T) {
	statuses := buildInfoSvrComponentStatuses(0, transaction.TransactionMgrStats{}, router.AdminSnapshot{})
	assertInfoComponentStatus(t, statuses, "infosvr.redis", "pending", false)
	assertInfoComponentStatus(t, statuses, "infosvr.transaction_mgr", "pending", false)
	assertInfoComponentStatus(t, statuses, "infosvr.router", "pending", false)
}
func findInfoComponentStatus(t *testing.T, statuses []bootstrap.ComponentStatus, name string) bootstrap.ComponentStatus {
	t.Helper()
	for _, status := range statuses {
		if status.Name == name {
			return status
		}
	}
	t.Fatalf("component %s not found", name)
	return bootstrap.ComponentStatus{}
}
func assertInfoComponentStatus(t *testing.T, statuses []bootstrap.ComponentStatus, name, state string, ready bool) {
	t.Helper()
	status := findInfoComponentStatus(t, statuses, name)
	if status.State != state || status.Ready != ready {
		t.Fatalf("unexpected component %s status: %+v", name, status)
	}
}
