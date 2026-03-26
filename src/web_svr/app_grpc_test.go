package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	websvrv1 "github.com/Iori372552686/GoOne/api/gen/web/websvr/v1"
	"github.com/Iori372552686/GoOne/common/gconf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestAppSvrImpl_StartGRPCServerSmoke(t *testing.T) {
	oldCfg := gconf.WebSvrCfg
	defer func() { gconf.WebSvrCfg = oldCfg }()

	port := reserveTCPPort(t)
	gconf.WebSvrCfg.HttpServer.AuthEnable = false
	gconf.WebSvrCfg.GRPCServer = gconf.GRPCServerConfig{
		Enabled: true,
		IP:      "127.0.0.1",
		Port:    port,
	}

	app := &AppSvrImpl{}
	if err := app.startGRPCServer(); err != nil {
		t.Fatalf("startGRPCServer err: %v", err)
	}
	defer func() {
		if app.grpcSrv != nil {
			app.grpcSrv.Stop()
		}
	}()

	target := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient err: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pingRsp websvrv1.PingRsp
	if err := conn.Invoke(ctx,
		"/web.websvr.v1.WebApiService/Ping",
		&websvrv1.PingReq{Msg: "smoke"},
		&pingRsp,
		grpc.WaitForReady(true),
	); err != nil {
		t.Fatalf("Ping invoke err: %v", err)
	}
	if pingRsp.GetMsg() != "pong: smoke" {
		t.Fatalf("Ping rsp.Msg = %q, want %q", pingRsp.GetMsg(), "pong: smoke")
	}

	healthRsp, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "web.websvr.v1.WebApiService",
	}, grpc.WaitForReady(true))
	if err != nil {
		t.Fatalf("Health Check err: %v", err)
	}
	if healthRsp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("Health status = %v, want SERVING", healthRsp.GetStatus())
	}

	stream, err := conn.NewStream(ctx,
		&grpc.StreamDesc{ServerStreams: true},
		"/web.websvr.v1.WebApiService/WatchPing",
		grpc.WaitForReady(true),
	)
	if err != nil {
		t.Fatalf("WatchPing NewStream err: %v", err)
	}
	if err := stream.SendMsg(&websvrv1.PingReq{Msg: "stream"}); err != nil {
		t.Fatalf("WatchPing SendMsg err: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("WatchPing CloseSend err: %v", err)
	}

	for i := 0; i < 3; i++ {
		var rsp websvrv1.PingRsp
		if err := stream.RecvMsg(&rsp); err != nil {
			t.Fatalf("WatchPing RecvMsg #%d err: %v", i+1, err)
		}
		if rsp.GetMsg() != "pong: stream" {
			t.Fatalf("WatchPing rsp[%d].Msg = %q, want %q", i, rsp.GetMsg(), "pong: stream")
		}
	}

	var last websvrv1.PingRsp
	if err := stream.RecvMsg(&last); err != io.EOF {
		t.Fatalf("WatchPing expected EOF, got %v", err)
	}
}

func reserveTCPPort(t *testing.T) int {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port err: %v", err)
	}
	defer lis.Close()

	return lis.Addr().(*net.TCPAddr).Port
}
