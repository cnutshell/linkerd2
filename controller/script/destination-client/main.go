package main

import (
	"context"
	"flag"
	"io"
	"math/rand"
	"time"

	pb "github.com/linkerd/linkerd2-proxy-api/go/destination"
	"github.com/linkerd/linkerd2/controller/destination"
	addrUtil "github.com/linkerd/linkerd2/pkg/addr"
	log "github.com/sirupsen/logrus"
)

// This is a throwaway script for testing the destination service

func main() {
	rand.Seed(time.Now().UnixNano())

	addr := flag.String("addr", ":8089", "address of destination service")
	path := flag.String("path", "strest-server.default.svc.cluster.local:8888", "destination path")
	flag.Parse()

	client, conn, err := destination.NewClient(*addr)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer conn.Close()

	req := &pb.GetDestination{
		Scheme: "k8s",
		Path:   *path,
	}

	rsp, err := client.Get(context.Background(), req)
	if err != nil {
		log.Fatal(err.Error())
	}

	for {
		update, err := rsp.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err.Error())
		}

		switch updateType := update.Update.(type) {
		case *pb.Update_Add:
			log.Println("Add:")
			log.Printf("labels: %v", updateType.Add.MetricLabels)
			for _, addr := range updateType.Add.Addrs {
				log.Printf("- %s:%d", addrUtil.ProxyIPToString(addr.Addr.GetIp()), addr.Addr.Port)
				log.Printf("  - labels: %v", addr.MetricLabels)
				switch addr.GetProtocolHint().GetProtocol().(type) {
				case *pb.ProtocolHint_H2_:
					log.Printf("  - protocol hint: H2")
				default:
					log.Printf("  - protocol hint: UNKNOWN")
				}
				switch identityType := addr.GetTlsIdentity().GetStrategy().(type) {
				case *pb.TlsIdentity_K8SPodIdentity_:
					log.Printf("  - pod identity: %s", identityType.K8SPodIdentity.PodIdentity)
					log.Printf("  - controller ns: %s", identityType.K8SPodIdentity.ControllerNs)
				}
			}
			log.Println()
		case *pb.Update_Remove:
			log.Println("Remove:")
			for _, addr := range updateType.Remove.Addrs {
				log.Printf("- %s:%d", addrUtil.ProxyIPToString(addr.GetIp()), addr.Port)
			}
			log.Println()
		case *pb.Update_NoEndpoints:
			log.Println("NoEndpoints:")
			log.Printf("- exists:%t", updateType.NoEndpoints.Exists)
			log.Println()
		}
	}
}
