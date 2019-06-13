package main

import (
	"context"
	"flag"

	"github.com/golang/glog"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type Runner func(s *mcclient.ClientSession)

var (
	optTimeout  int
	optDebug    bool
	optInsecure bool
	optCert     string
	optKey      string

	optAuthURL       string
	optUser          string
	optPass          string
	optDomain        string
	optProject       string
	optProjectDomain string

	optRegion       string
	optZone         string
	optEndpointType string

	runners = map[string]Runner{}
)

func registerRunner(name string, runner Runner) {
	runners[name] = runner
}

func init() {
	flag.IntVar(&optTimeout, "timeout", 5, "api request timeout")
	flag.BoolVar(&optInsecure, "insecure", true, "don't verify tls certificate")
	flag.BoolVar(&optDebug, "debug", false, "enable debug output")
	flag.StringVar(&optCert, "cert", "", "path to tls cert file")
	flag.StringVar(&optKey, "key", "", "path to tls key file")

	flag.StringVar(&optAuthURL, "auth-url", "", "keystone auth url")
	flag.StringVar(&optUser, "user", "", "keystone auth username")
	flag.StringVar(&optPass, "pass", "", "keystone auth password")
	flag.StringVar(&optDomain, "domain", "", "keystone auth domain")
	flag.StringVar(&optProject, "project", "", "keystone auth project")
	flag.StringVar(&optProjectDomain, "project-domain", "", "keystone auth project domain")

	flag.StringVar(&optRegion, "region", "", "keystone region")
	flag.StringVar(&optZone, "zone", "", "keystone zone")
	flag.StringVar(&optEndpointType, "endpoint-type", "", "keystone endpoint-type")
}

func main() {
	flag.Set("stderrthreshold", "0")
	flag.Set("logtostderr", "true")
	flag.Parse()

	client := mcclient.NewClient(
		optAuthURL,
		optTimeout,
		optDebug,
		optInsecure,
		optCert,
		optKey,
	)
	token, err := client.Authenticate(
		optUser,
		optPass,
		optDomain,
		optProject,
		optProjectDomain,
	)
	if err != nil {
		glog.Fatal(err)
	}
	s := client.NewSession(
		context.Background(),
		optRegion,
		optZone,
		optEndpointType,
		token,
		"v2",
	)

	for _, runner := range runners {
		runner(s)
	}
}
