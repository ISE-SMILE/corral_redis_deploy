package main

import (
	"context"
	"fmt"
	"github.com/ISE-SMILE/corral/services"
	"google.golang.org/grpc"
	"net"
	"os"
)

type DeploymentStrategy interface {
	Deploy(ctx context.Context, config *services.RedisDeploymentConfig) (*services.RedisClientConfig, error)
	Undeploy(ctx context.Context, config *services.RedisDeploymentConfig) error
}

type PluginServer struct {
	strategy DeploymentStrategy
}

func (p *PluginServer) Deploy(ctx context.Context, config *services.RedisDeploymentConfig) (*services.RedisClientConfig, error) {
	clientConfig, err := p.strategy.Deploy(ctx, config)
	if err != nil {
		return nil, err
	}

	return clientConfig, nil
}

func (p *PluginServer) Undeploy(ctx context.Context, config *services.RedisDeploymentConfig) (*services.Error, error) {
	err := p.strategy.Undeploy(ctx, config)
	if err != nil {
		msg := err.Error()
		return &services.Error{Message: &msg}, nil
	}

	return &services.Error{}, nil
}

func main() {
	//initialize the plugin backend
	p := PluginServer{}

	var mode string = "local"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	switch mode {
	case "local":
		p.strategy = &LocalRedisDeploymentStrategy{}
	case "k8s":
		fallthrough
	case "kubernetes":
		p.strategy = &KubernetesRedisDeploymentStrategy{}
	default:
		p.strategy = &LocalRedisDeploymentStrategy{}
	}

	//grab a random port
	lis, err := net.Listen("tcp", "localhost:0")
	defer lis.Close()
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()

	fmt.Printf("127.0.0.1:%d\n", lis.Addr().(*net.TCPAddr).Port)

	//start the service
	services.RegisterRedisDeploymentStrategyServer(server, &p)

	err = server.Serve(lis)
	if err != nil {
		fmt.Println("Corral Redis Deployment Plugin Stopped ")
	}

}
