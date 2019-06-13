package main

import (
	"flag"
	"time"

	"github.com/golang/glog"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func serverCreate(s *mcclient.ClientSession) *models.Server {
	inConfig := &compute.ServerConfigs{
		Disks: []*compute.DiskConfig{
			&compute.DiskConfig{
				ImageId: optServerImage,
			},
		},
	}
	in := &compute.ServerCreateInput{
		Name:          "server0",
		ServerConfigs: inConfig,
		VcpuCount:     1,
		VmemSize:      512,
		DisableDelete: options.Bool(false),
	}
	inJson := in.JSON(in)
	outJson, err := modules.Servers.Create(s, inJson)
	if err != nil {
		glog.Fatalf("server create: %v", err)
	}
	server := &models.Server{}
	if err := outJson.Unmarshal(server); err != nil {
		glog.Fatalf("server unmarshal: %v", err)
	}
	return server
}

func serverGet(s *mcclient.ClientSession, id string) (*models.Server, error) {
	serverJson, err := modules.Servers.Get(s, id, nil)
	if err != nil {
		return nil, err
	}
	server := &models.Server{}
	if err := serverJson.Unmarshal(server); err != nil {
		return nil, err
	}
	return server, nil
}

func serverDelete(s *mcclient.ClientSession, id string) {
	opts := &options.ServerDeleteOptions{
		OverridePendingDelete: options.Bool(true),
	}
	optsJson, _ := options.StructToParams(opts)
	_, err := modules.Servers.DeleteWithParam(s, id, optsJson, nil)
	if err != nil {
		glog.Fatalf("server delete: %v", err)
	}
}

func serverWaitStatuses(s *mcclient.ClientSession, id string, statuses []string) {
	var server *models.Server
	waiter := utils.NewFibonacciRetrierMaxElapse(
		time.Minute,
		func(r utils.FibonacciRetrier) (bool, error) {
			var err error
			server, err = serverGet(s, id)
			if err != nil {
				return false, err
			}
			status := server.Status
			glog.Infof("in status %s", status)
			for _, v := range statuses {
				if status == v {
					return true, nil
				}
			}
			return false, nil
		},
	)
	if _, err := waiter.Start(); err != nil {
		glog.Fatalf("wait status: %v", err)
	}
}

func serverWaitStatus(s *mcclient.ClientSession, id string, status string) {
	serverWaitStatuses(s, id, []string{status})
}

func serverStart(s *mcclient.ClientSession, id string) {
	_, err := modules.Servers.PerformAction(s, id, "start", nil)
	if err != nil {
		glog.Fatalf("perform start: %v", err)
	}
	serverWaitStatus(s, id, compute.VM_RUNNING)
}

func server(s *mcclient.ClientSession) {
	theServer := serverCreate(s)
	defer serverDelete(s, theServer.Id)
	serverWaitStatuses(s, theServer.Id, []string{
		compute.VM_READY,
		compute.VM_SCHEDULE_FAILED,
	})

	serverStart(s, theServer.Id)
	pressAnyKey()
}

var (
	optServer      bool
	optServerImage string
)

func init() {
	flag.BoolVar(&optServer, "server", false, "run example server code")
	flag.StringVar(&optServerImage, "server-image", "", "image id for rootfs")
	registerRunner("server", func(s *mcclient.ClientSession) {
		if optServer {
			server(s)
		}
	})
}
