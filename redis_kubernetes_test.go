package main

import (
	"context"
	"fmt"
	"github.com/ISE-SMILE/corral/services"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestKubeConfig(t *testing.T) {
	var port int = 30841
	krds := KubernetesRedisDeploymentStrategy{
		StorageClass: "zfs",
		NodePort:     &port,
	}

	conf, err := krds.config()

	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, conf)

	t.Log(conf)

}

func TestHelmClient(t *testing.T) {
	krds := KubernetesRedisDeploymentStrategy{}
	_, _, err := krds.helmClient()
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelmAddRepo(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	krds := KubernetesRedisDeploymentStrategy{}
	settings, _, err := krds.helmClient()
	if err != nil {
		t.Fatal(err)
	}

	err = addHelmRepo(settings)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKubernetesDeployment(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	var port int = 30841
	krds := KubernetesRedisDeploymentStrategy{
		Namespace:    "sw-experimentes",
		StorageClass: "zfs",
		NodePort:     &port,
	}

	var ctx context.Context
	var cancleFunc context.CancelFunc
	if deadline, ok := t.Deadline(); ok {
		ctx, cancleFunc = context.WithTimeout(context.Background(), deadline.Sub(time.Now()))
	} else {
		ctx, cancleFunc = context.WithTimeout(context.Background(), 5*time.Minute)
	}
	cnf := services.RedisDeploymentConfig{}

	ccnf, err := krds.Deploy(ctx, &cnf)
	defer cancleFunc()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(fmt.Sprintf("%v", ccnf))
}
