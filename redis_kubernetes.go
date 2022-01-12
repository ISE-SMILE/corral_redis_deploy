package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ISE-SMILE/corral/services"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var RedisKubernetesChartRepo = repo.Entry{
	Name: "groundhog2k",
	URL:  "https://groundhog2k.github.io/helm-charts/",
}

const RedisKubernetesChart = "groundhog2k/redis"
const RedisKubernetesDeploymentName = "corral-redis"

type KubernetesRedisDeploymentStrategy struct {
	Namespace    string
	StorageClass string
	NodePort     *int

	deploymentName string

	//helmClient helmClient.Client
}

type kubernetesRedisConfig struct {
	Service struct {
		Type     string `yaml:"type,omitempty"`
		NodePort *int   `yaml:"nodePort,omitempty"`
	} `yaml:"service,omitempty"`

	Storage struct {
		Class         string `yaml:"className,omitempty"`
		RequestedSize string `yaml:"requestedSize,omitempty"`
	} `yaml:"storage,omitempty"`

	Resources struct {
		Limits struct {
			Memory string `yaml:"memory,omitempty"`
		} `yaml:"limits,omitempty"`
	} `yaml:"resources,omitempty"`
}

func (k *KubernetesRedisDeploymentStrategy) config() (map[string]interface{}, error) {
	conf := &kubernetesRedisConfig{}

	conf.Resources.Limits.Memory = "512Mi"
	conf.Storage.Class = k.StorageClass

	if k.NodePort != nil {
		conf.Service.Type = "NodePort"
		conf.Service.NodePort = k.NodePort
	}

	//and just so we get the right type at the end...
	buf := bytes.NewBuffer(make([]byte, 0))
	enc := yaml.NewEncoder(buf)
	err := enc.Encode(conf)
	if err != nil {
		return nil, err
	}
	var vals map[string]interface{}
	err = yaml.NewDecoder(buf).Decode(&vals)
	if err != nil {
		return nil, err
	}

	return vals, nil
}
func (k *KubernetesRedisDeploymentStrategy) Deploy(ctx context.Context, config *services.RedisDeploymentConfig) (*services.RedisClientConfig, error) {
	settings, actionConfig, err := k.helmClient()
	if err != nil {
		return nil, err
	}

	err = addHelmRepo(settings)
	if err != nil {
		return nil, err
	}

	// define values
	vals, err := k.config()
	if err != nil {
		return nil, err
	}

	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = true
	releases, err := listAction.Run()
	if err != nil {
		return nil, err
	}
	var exsists = false
	for _, release := range releases {
		if release.Name == RedisKubernetesDeploymentName {
			exsists = true
			break
		}
	}

	if !exsists {
		client := action.NewInstall(actionConfig)

		cp, err := client.ChartPathOptions.LocateChart(RedisKubernetesChart, settings)
		if err != nil {
			log.Fatal(err)
		}

		// Check chart dependencies to make sure all are present in /charts
		chart, err := loader.Load(cp)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		client.Namespace = k.Namespace
		client.ReleaseName = RedisKubernetesDeploymentName

		// install the chart here
		rel, err := client.Run(chart, vals)
		if err != nil {
			return nil, err
		}
		log.Printf("Installed Chart from path: %s in namespace: %s\n", rel.Name, rel.Namespace)
		log.Println(rel.Config)
	}

	redis_conf := services.RedisClientConfig{
		Addrs: nil,
	}
	k8sClient, k8sConfig, err := mkK8sClient(settings)
	if err != nil {
		return nil, err
	}

	if k.NodePort != nil {
		host := strings.Split(k8sConfig.Host[len("https://"):], ":")[0]
		redis_conf.Addrs = []string{
			fmt.Sprintf("%s:%d", host, *k.NodePort),
		}
	} else {
		//well that looks terrible..
		get, err := k8sClient.CoreV1().Services(k.Namespace).Get(ctx, fmt.Sprintf("%s-master", RedisKubernetesDeploymentName), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		redis_conf.Addrs = []string{
			fmt.Sprintf("%s:%d", get.Spec.ClusterIP, 6379),
		}
	}

	return &redis_conf, err
}

func mkK8sClient(settings *cli.EnvSettings) (*kubernetes.Clientset, *rest.Config, error) {
	buildConfig := settings.KubeConfig
	if settings.KubeConfig != "" {
		buildConfig = settings.KubeConfig
	} else if home := homedir.HomeDir(); home != "" {
		buildConfig = filepath.Join(home, ".kube", "config")
	} else {
		return nil, nil, fmt.Errorf("failed to locate k8s config")
	}

	k8s, err := clientcmd.BuildConfigFromFlags("", buildConfig)
	if err != nil {
		return nil, nil, err
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(k8s)
	if err != nil {
		return nil, nil, err
	}
	return clientset, k8s, nil
}

func (k *KubernetesRedisDeploymentStrategy) helmClient() (*cli.EnvSettings, *action.Configuration, error) {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err := actionConfig.Init(settings.RESTClientGetter(), k.Namespace,
		os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Debugf("%+v", err)
		return nil, nil, err
	}
	return settings, actionConfig, nil
}

func (k *KubernetesRedisDeploymentStrategy) Undeploy(ctx context.Context, config *services.RedisDeploymentConfig) error {
	_, actionConfig, err := k.helmClient()
	if err != nil {
		return err
	}

	client := action.NewUninstall(actionConfig)
	msg, err := client.Run(RedisKubernetesDeploymentName)
	if err != nil {
		return err
	}

	log.Println(msg.Info)
	return nil
}

func addHelmRepo(settings *cli.EnvSettings) error {
	// find out if we need to update the repo first
	b, err := ioutil.ReadFile(settings.RepositoryConfig)
	if err != nil && !os.IsNotExist(err) {
		log.Debugf("could not load helm repo file %v", err)
		return err
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		log.Debugf("could not read helm repo file %v", err)
		return err
	}

	//check if we need to install the repp
	if !f.Has(RedisKubernetesChartRepo.Name) {
		r, err := repo.NewChartRepository(&RedisKubernetesChartRepo, getter.All(settings))
		if err != nil {
			return err
		}

		if _, err := r.DownloadIndexFile(); err != nil {
			return fmt.Errorf("looks like %q is not a valid chart repository or cannot be reached", RedisKubernetesChartRepo.URL)
		}

		f.Update(&RedisKubernetesChartRepo)

		if err := f.WriteFile(settings.RepositoryConfig, 0644); err != nil {
			return err
		}
	}
	return nil
}
