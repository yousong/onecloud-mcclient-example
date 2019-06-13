package main

import (
	"flag"
	"time"

	"github.com/golang/glog"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func lbCreate(s *mcclient.ClientSession) *models.Loadbalancer {
	lbCreateOpt := &options.LoadbalancerCreateOptions{
		NAME:    "lb0",
		Network: optLbNetwork,
	}
	params, _ := options.StructToParams(lbCreateOpt)
	json, err := modules.Loadbalancers.Create(s, params)
	if err != nil {
		glog.Fatalf("create lb: %v\n", err)
	}
	lb := &models.Loadbalancer{}
	err = json.Unmarshal(lb)
	if err != nil {
		glog.Fatalf("unmarshal lb: %v", err)
	}
	return lb
}

func lbDelete(s *mcclient.ClientSession, lb *models.Loadbalancer) {
	_, err := modules.Loadbalancers.Delete(s, lb.Id, nil)
	if err != nil {
		glog.Errorf("delete lb: %v", err)
	}
}

func lblisCreate(s *mcclient.ClientSession, lb *models.Loadbalancer, lbbg *models.LoadbalancerBackendGroup) *models.LoadbalancerListener {
	lblisCreateOpt := &options.LoadbalancerListenerCreateOptions{
		NAME:         "lb0lis0",
		Loadbalancer: lb.Id,
		ListenerType: "tcp",
		ListenerPort: options.Int(80),
		Scheduler:    "rr",
		BackendGroup: lbbg.Id,
	}
	params, _ := options.StructToParams(lblisCreateOpt)
	json, err := modules.LoadbalancerListeners.Create(s, params)
	if err != nil {
		glog.Fatalf("create lblis: %v\n", err)
	}
	lblis := &models.LoadbalancerListener{}
	err = json.Unmarshal(lblis)
	if err != nil {
		glog.Fatalf("unmarshal lblis: %v", err)
	}
	return lblis
}

func lblisDelete(s *mcclient.ClientSession, lblis *models.LoadbalancerListener) {
	_, err := modules.LoadbalancerListeners.Delete(s, lblis.Id, nil)
	if err != nil {
		glog.Errorf("delete lblis: %v", err)
	}
}

func lbbgCreate(s *mcclient.ClientSession, lb *models.Loadbalancer) *models.LoadbalancerBackendGroup {
	lbbgCreateOpt := &options.LoadbalancerBackendGroupCreateOptions{
		NAME:         "lb0bg0",
		LOADBALANCER: lb.Id, // use Loadbalancer
	}
	params, _ := options.StructToParams(lbbgCreateOpt)
	json, err := modules.LoadbalancerBackendGroups.Create(s, params)
	if err != nil {
		glog.Fatalf("create lbbg: %v\n", err)
	}
	lbbg := &models.LoadbalancerBackendGroup{}
	err = json.Unmarshal(lbbg)
	if err != nil {
		glog.Fatalf("unmarshal lbbg: %v", err)
	}
	return lbbg
}

func lbbgDelete(s *mcclient.ClientSession, lbbg *models.LoadbalancerBackendGroup) {
	_, err := modules.LoadbalancerBackendGroups.Delete(s, lbbg.Id, nil)
	if err != nil {
		glog.Errorf("delete lbbg: %v", err)
	}
}

func lbbCreate(s *mcclient.ClientSession, lbbg *models.LoadbalancerBackendGroup, ip string, port int, weight int) *models.LoadbalancerBackend {
	lbbCreateOpt := &options.LoadbalancerBackendCreateOptions{
		BackendGroup: lbbg.Id,
		BackendType:  "ip",
		Backend:      ip,
		Port:         &port,
		Weight:       &weight,
	}
	params, _ := options.StructToParams(lbbCreateOpt)
	json, err := modules.LoadbalancerBackends.Create(s, params)
	if err != nil {
		glog.Fatalf("create lbb: %v\n", err)
	}
	lbb := &models.LoadbalancerBackend{}
	err = json.Unmarshal(lbb)
	if err != nil {
		glog.Fatalf("unmarshal lbb: %v", err)
	}
	return lbb
}

func lbbDelete(s *mcclient.ClientSession, lbb *models.LoadbalancerBackend) {
	_, err := modules.LoadbalancerBackends.Delete(s, lbb.Id, nil)
	if err != nil {
		glog.Errorf("delete lbb: %v", err)
	}
}

func lb(s *mcclient.ClientSession) {
	lb := lbCreate(s)
	defer lbDelete(s, lb)

	lbbg := lbbgCreate(s, lb)
	defer lbbgDelete(s, lbbg)

	lblis := lblisCreate(s, lb, lbbg)
	defer func() {
		lblisDelete(s, lblis)
		// XXX async delete.  wait so that refcount of lbbg decr to 0
		time.Sleep(200 * time.Millisecond)
	}()

	lbb0 := lbbCreate(s, lbbg, "192.168.3.133", 8088, 1)
	defer lbbDelete(s, lbb0)
	lbb1 := lbbCreate(s, lbbg, "192.168.3.233", 8188, 2)
	defer lbbDelete(s, lbb1)

	{ // list backend by listener, i.e. vip:vport
		listOpt := &options.LoadbalancerBackendListOptions{
			BackendGroup: lblis.BackendGroupId,
		}
		params, _ := options.StructToParams(listOpt)
		lr, err := modules.LoadbalancerBackends.List(s, params)
		if err != nil {
			glog.Fatalf("list lbb: %v", err)
		}
		glog.Infoln(jsonutils.Marshal(lr).PrettyString())
	}
	{ // list backend by rip:rport
		listOpt := &options.LoadbalancerBackendListOptions{
			Address: "192.168.3.133",
			Port:    options.Int(8088),
		}
		params, _ := options.StructToParams(listOpt)
		lr, err := modules.LoadbalancerBackends.List(s, params)
		if err != nil {
			glog.Fatalf("list lbb: %v", err)
		}
		glog.Infoln(jsonutils.Marshal(lr).PrettyString())
	}
}

var (
	optLbNetwork string
	optLb        bool
)

func init() {
	flag.BoolVar(&optLb, "lb", false, "run example lb code")
	flag.StringVar(&optLbNetwork, "lb-network", "", "network to allocate virtual ip for loadbalancer")
	registerRunner("lb", func(s *mcclient.ClientSession) {
		if optLb {
			lb(s)
		}
	})
}
