package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tensorflow/k8s/pkg/controller"
	"github.com/tensorflow/k8s/pkg/spec"
	"github.com/tensorflow/k8s/pkg/util"
	"github.com/tensorflow/k8s/pkg/util/k8sutil"
	"github.com/tensorflow/k8s/version"

	"github.com/ghodss/yaml"
	log "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	election "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

var (
	namespace  string
	name       string
	gcInterval time.Duration

	chaosLevel           int
	controllerConfigFile string
	printVersion         bool
	grpcServerFile       string
)

var (
	controllerConfig = spec.ControllerConfig{}
	leaseDuration    = 15 * time.Second
	renewDuration    = 5 * time.Second
	retryPeriod      = 3 * time.Second
	tfJobClient      *k8sutil.TfJobRestClient
)

func init() {
	// chaos level will be removed once we have a formal tool to inject failures.
	flag.IntVar(&chaosLevel, "chaos-level", -1, "DO NOT USE IN PRODUCTION - level of chaos injected into the TfJob created by the operator.")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.DurationVar(&gcInterval, "gc-interval", 10*time.Minute, "GC interval")
	flag.StringVar(&controllerConfigFile, "controller_config_file", "", "Path to file containing the controller config.")
	flag.Parse()

	// Workaround for watching TPR resource.
	restCfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		panic(err)
	}
	controller.MasterHost = restCfg.Host
	tfJobClient, err = k8sutil.NewTfJobClient()
	if err != nil {
		panic(err)
	}
	controller.KubeHttpCli = tfJobClient.Client()

	if controllerConfigFile != "" {
		log.Infof("Loading controller config from %v.", controllerConfigFile)

		data, err := ioutil.ReadFile(controllerConfigFile)
		if err != nil {
			log.Fatalf("Could not read file: %v. Error: %v", controllerConfigFile, err)
		}

		err = yaml.Unmarshal(data, &controllerConfig)
		if err != nil {
			log.Fatalf("Could not parse controller config; Error: %v\n", err)
		}

		log.Infof("ControllerConfig: %v", util.Pformat(controllerConfig))
	} else {
		log.Info("No controller_config_file provided; using empty config.")
	}

}

func main() {
	namespace = os.Getenv("MY_POD_NAMESPACE")
	if len(namespace) == 0 {
		log.Fatalf("must set env MY_POD_NAMESPACE")
	}
	name = os.Getenv("MY_POD_NAME")
	if len(name) == 0 {
		log.Fatalf("must set env MY_POD_NAME")
	}

	setupSignalHandler()
	version.PrintVersion(printVersion)

	id, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	rl := &resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "tf-operator",
		},
		Client: k8sutil.MustNewKubeClient().CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: &record.FakeRecorder{},
		},
	}

	election.RunOrDie(election.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				log.Fatalf("leader election lost")
			},
		},
	})

	panic("unreachable")
}

func setupSignalHandler() {
	shutdownSignals := []os.Signal{os.Interrupt, syscall.SIGTERM}
	c := make(chan os.Signal, 1)
	signal.Notify(c, shutdownSignals...)
	go func() {
		log.Infof("received signal: %v", <-c)
		os.Exit(1)
	}()
}

func run(stop <-chan struct{}) {
	kubeCli := k8sutil.MustNewKubeClient()

	// TODO(jlewi): Should we start chaos?
	// startChaos(context.Background(), cfg.KubeCli, cfg.Namespace, chaosLevel)
	apiCli := k8sutil.MustNewApiExtensionsClient()

	for {
		c := controller.New(kubeCli, apiCli, tfJobClient, namespace, controllerConfig)
		err := c.Run()
		switch err {
		case controller.ErrVersionOutdated:
		default:
			log.Fatalf("controller Run() ended with failure: %v", err)
		}
	}
}

// TODO(jlewi): We should add chaos back.
//func startChaos(ctx context.Context, kubecli kubernetes.Interface, ns string, chaosLevel int) {
//  m := chaos.NewMonkeys(kubecli)
//  ls := labels.SelectorFromSet(map[string]string{"app": "etcd"})
//
//  switch chaosLevel {
//  case 1:
//    log.Info("chaos level = 1: randomly kill one etcd pod every 30 seconds at 50%")
//    c := &chaos.CrashConfig{
//      Namespace: ns,
//      Selector:  ls,
//
//      KillRate:        rate.Every(30 * time.Second),
//      KillProbability: 0.5,
//      KillMax:         1,
//    }
//    go func() {
//      time.Sleep(60 * time.Second) // don't start until quorum up
//      m.CrushPods(ctx, c)
//    }()
//
//  case 2:
//    log.Info("chaos level = 2: randomly kill at most five etcd pods every 30 seconds at 50%")
//    c := &chaos.CrashConfig{
//      Namespace: ns,
//      Selector:  ls,
//
//      KillRate:        rate.Every(30 * time.Second),
//      KillProbability: 0.5,
//      KillMax:         5,
//    }
//
//    go m.CrushPods(ctx, c)
//
//  default:
//  }
//}
