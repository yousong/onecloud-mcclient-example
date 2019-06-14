package main

import (
	"flag"
	"path/filepath"
	"time"

	"github.com/golang/glog"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	apis "yunion.io/x/onecloud/pkg/apis/ansible"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ansible"
	//"yunion.io/x/onecloud/pkg/mcclient/options"
)

// TODO
// - get user, pass, tenant name
// - get rpms
//

const (
	lbagent_conf = `
region = '{{ lb_region }}'
auth_uri = 'http://10.168.222.136:35357/v3'
admin_user = 'regionadmin'
admin_password = 'xxxxxxxxxxxxxxxx'
admin_tenant_name = 'system'

data_preserve_n = 10
base_data_dir = "/opt/cloud/workspace/lbagent"

api_lbagent_id = 'xxxxxxxx'
api_lbagent_hb_interval = 60

api_sync_interval = 5
api_list_batch_size = 2048
`
)

func ansiblePbCreate(s *mcclient.ClientSession, server *models.Server) *models.AnsiblePlaybook {
	input := &apis.AnsiblePlaybookCreateInput{
		Name: "pb0",
		Playbook: ansible.Playbook{
			Inventory: ansible.Inventory{
				Hosts: []ansible.Host{
					{
						Name: server.Id,
						Vars: map[string]string{
							"lb_region":      "YunionHQ",
							"ansible_become": "yes",
						},
					},
				},
			},
			Modules: []ansible.Module{
				{
					Name: "group",
					Args: []string{
						"name=yunion",
						"state=present",
					},
				},
				{
					Name: "user",
					Args: []string{
						"name=yunion",
						"state=present",
						"group=yunion",
					},
				},
				{
					Name: "file",
					Args: []string{
						"path=/etc/yunion",
						"state=directory",
						"owner=yunion",
						"group=yunion",
						"mode=755",
					},
				},
				{
					Name: "template",
					Args: []string{
						"src=lbagent_conf",
						"dest=/etc/yunion/lbagent.conf",
						"owner=yunion",
						"group=yunion",
						"mode=600",
					},
				},
				//{
				//Name: "command",
				//Args: []string{
				//"sleep 1",
				//},
				//},
			},
			Files: map[string][]byte{
				"lbagent_conf": []byte(lbagent_conf),
			},
		},
	}
	{
		// glob for rpms
		basenames := []string{
			"packages/telegraf",
			"packages/gobetween",
			"packages/keepalived",
			"packages/haproxy",
			"updates/yunion-lbagent",
		}
		mods := []ansible.Module{}
		for _, basename := range basenames {
			pattern := filepath.Join("/opt/yunion/upgrade/rpms", basename+"-*.rpm")
			matches, err := filepath.Glob(pattern)
			if err != nil {
				glog.Fatalf("glob error %s: %v", pattern, err)
			}
			if len(matches) == 0 {
				glog.Fatalf("glob nomatch %s", pattern)
			}
			path := matches[len(matches)-1]
			name := filepath.Base(path)
			destPath := filepath.Join("/tmp", name)
			mods = append(mods,
				ansible.Module{
					Name: "copy",
					Args: []string{
						"src=" + path,
						"dest=" + destPath,
					},
				},
				ansible.Module{
					Name: "yum",
					Args: []string{
						"name=" + destPath,
						"state=installed",
						"update_cache=yes",
					},
				},
				ansible.Module{
					Name: "file",
					Args: []string{
						"name=" + destPath,
						"state=absent",
					},
				},
			)
		}
		input.Playbook.Modules = append(input.Playbook.Modules, mods...)
		input.Playbook.Modules = append(input.Playbook.Modules,
			ansible.Module{
				Name: "copy",
				Args: []string{
					"remote_src=yes",
					"src=/opt/yunion/share/lbagent/yunion-lbagent.service",
					"dest=/etc/systemd/system/yunion-lbagent.service",
				},
			},
			ansible.Module{
				Name: "systemd",
				Args: []string{
					"name=yunion-lbagent",
					"enabled=yes",
					"state=started",
					"daemon_reload=yes",
				},
			},
		)
	}
	params := input.JSON(input)
	pbJson, err := modules.AnsiblePlaybooks.Create(s, params)
	if err != nil {
		glog.Fatalf("create ansibleplaybook: %v", err)
	}
	pb := &models.AnsiblePlaybook{}
	if err := pbJson.Unmarshal(pb); err != nil {
		glog.Fatalf("unmarshal ansibleplaybook: %v", err)
	}
	return pb
}

func ansiblePbDelete(s *mcclient.ClientSession, id string) {
	_, err := modules.AnsiblePlaybooks.Delete(s, id, nil)
	if err != nil {
		glog.Fatalf("delete ansibleplaybook: %v", err)
	}
}

func ansiblePbGet(s *mcclient.ClientSession, id string) (*models.AnsiblePlaybook, error) {
	pbJson, err := modules.AnsiblePlaybooks.Get(s, id, nil)
	if err != nil {
		return nil, err
	}
	pb := &models.AnsiblePlaybook{}
	if err := pbJson.Unmarshal(pb); err != nil {
		return nil, err
	}
	return pb, nil
}

func ansiblePbStop(s *mcclient.ClientSession, id string) (*models.AnsiblePlaybook, error) {
	pbJson, err := modules.AnsiblePlaybooks.PerformAction(s, id, "stop", nil)
	if err != nil {
		return nil, err
	}
	pb := &models.AnsiblePlaybook{}
	if err := pbJson.Unmarshal(pb); err != nil {
		return nil, err
	}
	return pb, nil
}

func ansiblePbWaitStatuses(s *mcclient.ClientSession, id string, statuses []string) *models.AnsiblePlaybook {
	var pb *models.AnsiblePlaybook
	waiter := utils.NewFibonacciRetrierMaxElapse(
		time.Minute,
		func(r utils.FibonacciRetrier) (bool, error) {
			var err error
			pb, err = ansiblePbGet(s, id)
			if err != nil {
				return false, err
			}
			status := pb.Status
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
		ansiblePbStop(s, id)
		glog.Fatalf("wait status: %v", err)
	}
	return pb
}

func ansiblePb(s *mcclient.ClientSession) {
	server, err := serverGet(s, optAnsiblePbServer)
	if err != nil {
		glog.Fatalf("get server: %v", err)
	}
	pb := ansiblePbCreate(s, server)
	defer ansiblePbDelete(s, pb.Id)

	pb = ansiblePbWaitStatuses(s, pb.Id, []string{
		"failed",
		"canceled",
		"succeeded",
		"unknown",
	})
	if pb != nil {
		j := jsonutils.Marshal(pb)
		glog.Infof("%s", j.PrettyString())
	}
}

var (
	optAnsiblePb       bool
	optAnsiblePbServer string
)

func init() {
	flag.BoolVar(&optAnsiblePb, "ansible-pb", false, "run example ansible-pb code")
	flag.StringVar(&optAnsiblePbServer, "ansible-pb-server", "", "name or id of server ansible to operate on")

	registerRunner("ansible-pb", func(s *mcclient.ClientSession) {
		if optAnsiblePb {
			ansiblePb(s)
		}
	})
}
