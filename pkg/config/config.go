package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/openshift-pipelines/release-tests/pkg/assert"
)

const (
	// APIRetry defines the frequency at which we check for updates against the
	// k8s api when waiting for a specific condition to be true.
	APIRetry = time.Second * 5

	// APITimeout defines the amount of time we should spend querying the k8s api
	// when waiting for a specific condition to be true.
	APITimeout = time.Minute * 20
	// Timeout httpClient
	Timeout = time.Second * 10

	// ConsistentlyDuration sets  the default duration for Consistently. Consistently will verify that your condition is satisfied for this long.
	ConsistentlyDuration = 30 * time.Second

	//TektonConfigName specify the name of tekton config
	TektonConfigName = "config"

	//TargetNamespace specify the name of Target namespace
	TargetNamespace = "openshift-pipelines"

	// Name of the pipeline controller deployment
	PipelineControllerName = "tekton-pipelines-controller"
	PipelineControllerSA   = "tekton-pipelines-controller"

	PipelineWebhookName          = "tekton-pipelines-webhook"
	PipelineWebhookConfiguration = "webhook.tekton.dev"
	SccAnnotationKey             = "operator.tekton.dev"

	// Name of the trigger deployment
	TriggerControllerName = "tekton-triggers-controller"
	TriggerWebhookName    = "tekton-triggers-webhook"

	// Default config for auto pruner
	PrunerSchedule = "0 8 * * *"
	PrunerNamePrefix = "tekton-resource-pruner-"
)

// Flags holds the command line flags or defaults for settings in the user's environment.
// See EnvironmentFlags for a list of supported fields
// Todo: change initialization of falgs when required by parsing them or from environment variable
var Flags = initializeFlags()

// EnvironmentFlags define the flags that are needed to run the e2e tests.
type EnvironmentFlags struct {
	Cluster          string // K8s cluster (defaults to cluster in kubeconfig)
	Kubeconfig       string // Path to kubeconfig (defaults to ./kube/config)
	DockerRepo       string // Docker repo (defaults to $KO_DOCKER_REPO)
	CSV              string // Default csv openshift-pipelines-operator.v0.9.1
	Channel          string // Default channel canary
	CatalogSource    string
	SubscriptionName string
	InstallPlan      string // Default Installationplan Automatic
	OperatorVersion  string
	TknVersion       string
}

func initializeFlags() *EnvironmentFlags {
	var f EnvironmentFlags
	flag.StringVar(&f.Cluster, "cluster", "",
		"Provide the cluster to test against. Defaults to the current cluster in kubeconfig.")

	var defaultKubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		defaultKubeconfig = os.Getenv("KUBECONFIG")
	} else if usr, err := user.Current(); err == nil {
		defaultKubeconfig = path.Join(usr.HomeDir, ".kube/config")
	}

	flag.StringVar(&f.Kubeconfig, "kubeconfig", defaultKubeconfig,
		"Provide the path to the `kubeconfig` file you'd like to use for these tests. The `current-context` will be used.")

	defaultRepo := os.Getenv("KO_DOCKER_REPO")
	flag.StringVar(&f.DockerRepo, "dockerrepo", defaultRepo,
		"Provide the uri of the docker repo you have uploaded the test image to using `uploadtestimage.sh`. Defaults to $KO_DOCKER_REPO")

	defaultChannel := os.Getenv("CHANNEL")
	flag.StringVar(&f.Channel, "channel", defaultChannel,
		"Provide channel to subcribe your operator you'd like to use for these tests. By default `canary` will be used.")

	defaultCatalogSource := os.Getenv("CATALOG_SOURCE")
	flag.StringVar(&f.CatalogSource, "catalogsource", defaultCatalogSource,
		"Provide defaultCatalogSource to subscribe operator from. By default `pre-stage-operators` will be used.")

	defaultSubscriptionName := os.Getenv("SUBSCRIPTION_NAME")
	flag.StringVar(&f.SubscriptionName, "subscriptionName", defaultSubscriptionName,
		"Provide defaultSubscriptionName to operator, By default `openshift-pipelines-operator-rh` will be used.")

	defaultPlan := os.Getenv("INSTALL_PLAN")
	flag.StringVar(&f.InstallPlan, "installplan", defaultPlan,
		"Provide Install Approval plan for your operator you'd like to use for these tests. By default `Automatic` will be used.")

	defaultOpVersion := os.Getenv("CSV_VERSION")
	flag.StringVar(&f.OperatorVersion, "opversion", defaultOpVersion,
		"Provide Operator version for your operator you'd like to use for these tests. By default `v0.9.1` ")

	defaultCsv := os.Getenv("CSV")
	flag.StringVar(&f.CSV, "csv", defaultCsv+defaultOpVersion,
		"Provide csv for your operator you'd like to use for these tests. By default `openshift-pipelines-operator.v0.9.1` will be used.")

	defaultTkn := os.Getenv("TKN_VERSION")
	flag.StringVar(&f.TknVersion, "tknversion", defaultTkn,
		"Provide tknversion to download specified cli binary you'd like to use for these tests. By default `0.6.0` will be used.")

	return &f
}

func Dir() string {
	_, b, _, _ := runtime.Caller(0)
	configDir := path.Join(path.Dir(b), "..", "..", "template")
	return configDir
}

func File(elem ...string) string {
	path := append([]string{Dir()}, elem...)
	return filepath.Join(path...)
}

func Read(path string) ([]byte, error) {
	return ioutil.ReadFile(File(path))
}

func TempDir() (string, error) {
	tmp := filepath.Join(Dir(), "..", "tmp")
	if _, err := os.Stat(tmp); os.IsNotExist(err) {
		err := os.Mkdir(tmp, 0755)
		return tmp, err
	}
	return tmp, nil
}

func TempFile(elem ...string) (string, error) {
	tmp, err := TempDir()
	if err != nil {
		return "", err
	}
	path := append([]string{tmp}, elem...)
	return filepath.Join(path...), nil
}

func RemoveTempDir() {
	var err error
	tmp, _ := TempDir()
	err = os.RemoveAll(tmp)
	assert.NoError(err, fmt.Sprintf("Error: In deleting directory: %+v ", tmp))
}

func Path(elem ...string) string {
	td := filepath.Join(Dir(), "..")
	if _, err := os.Stat(td); os.IsNotExist(err) {
		assert.NoError(err, fmt.Sprintf("Error: in identifying test data path %+v ", td))
	}
	return filepath.Join(append([]string{td}, elem...)...)
}

func ReadBytes(elem string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(Path(elem))
	if err != nil {
		return nil, fmt.Errorf("couldn't load test data example PullRequest event data: %v", err)
	}
	return bytes, nil
}

// ResourceNames holds names of various resources.
type ResourceNames struct {
	TektonPipeline  string
	TektonTrigger   string
	TektonDashboard string
	TektonAddon     string
	TektonConfig    string
	Namespace       string
	TargetNamespace string
}
