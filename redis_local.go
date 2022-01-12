package main

import (
	"context"
	"fmt"
	"github.com/ISE-SMILE/corral/services"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const redisDockerImage = "redis:6.2.4-alpine"
const redisContainerName = "corral_redis"

type LocalRedisDeploymentStrategy struct {
	containerID string
	port        string
}

func (l *LocalRedisDeploymentStrategy) Deploy(config *services.RedisDeploymentConfig) (*services.RedisClientConfig, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	_, err = cli.ImagePull(ctx, redisDockerImage, types.ImagePullOptions{})
	if err != nil {
		return nil, err
	}

	candiates, err := cli.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("name", redisContainerName)),
		All:     true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to check running containers")
	}

	var id string
	if len(candiates) > 0 {
		// the container alread exsists?
		id = candiates[0].ID

		//now we start the container afterwards but this way it will not crash :P
		if candiates[0].State != "running" {
			err := cli.ContainerRestart(ctx, id, nil)
			if err != nil {
				return nil, fmt.Errorf("found an exsiting local instance but could not restart it, %+v", err)
			}
		}

	} else {

		cc, err := cli.ContainerCreate(ctx,
			&container.Config{
				Image: redisDockerImage,
				ExposedPorts: nat.PortSet{
					"6379/tcp": struct{}{},
				},
				Tty: false,
			},
			&container.HostConfig{
				PortBindings: nat.PortMap{
					"6379/tcp": []nat.PortBinding{
						{
							HostIP: "0.0.0.0",
						},
					},
				},
			},

			nil, nil, redisContainerName)

		if err != nil {
			return nil, err
		}
		id = cc.ID
	}

	err = cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}
	container, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup local port")
	}

	if container.NetworkSettings == nil {
		return nil, fmt.Errorf("failed to lookup local port")
	}
	if ports, ok := container.NetworkSettings.Ports["6379/tcp"]; ok {
		l.port = ports[0].HostPort
	} else {
		return nil, fmt.Errorf("failed to lookup local port")
	}

	l.containerID = id

	return &services.RedisClientConfig{
		Addrs:          []string{fmt.Sprintf(":%+v", l.port)},
		DB:             0,
		User:           "",
		Password:       "",
		RouteByLatency: false,
		RouteRandomly:  false,
	}, nil
}

func (l *LocalRedisDeploymentStrategy) Undeploy(config *services.RedisDeploymentConfig) error {
	if l.containerID == "" {
		return fmt.Errorf("redis was not deployed")
	}

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	err = cli.ContainerRemove(ctx, l.containerID, types.ContainerRemoveOptions{
		Force: true,
	})

	l.containerID = ""
	l.port = ""

	return err
}
