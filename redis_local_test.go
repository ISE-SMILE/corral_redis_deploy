package main

import (
	"context"
	redis "github.com/go-redis/redis/v8"
	"testing"
)

func TestLocalRedisDeploymentStrategy(t *testing.T) {
	lrds := &LocalRedisDeploymentStrategy{}

	conf, err := lrds.Deploy(nil)

	if err != nil {
		t.Fatalf("failed to deploy, %+v", err)
	}

	t.Logf("%+v", conf)

	err = lrds.Undeploy(nil)
	if err != nil {
		t.Fatalf("failed to undeploy, %+v", err)
	}

}

func TestLocalRedisDeploymentStrategy_WithConnection(t *testing.T) {
	lrds := &LocalRedisDeploymentStrategy{}

	conf, err := lrds.Deploy(nil)

	if err != nil {
		t.Fatalf("failed to deploy, %+v", err)
	}

	cli := redis.NewClient(&redis.Options{
		Addr:     conf.Addrs[0],
		Username: conf.User,
		Password: conf.Password,
		DB:       int(conf.DB),
	})
	defer cli.Close()
	pong, err := cli.Ping(context.Background()).Result()
	if err != nil {
		t.Fatalf("failed to connect, %+v", err)
	}

	t.Logf("%+v - %+v", conf, pong)

	err = lrds.Undeploy(nil)
	if err != nil {
		t.Fatalf("failed to undeploy, %+v", err)
	}
}
