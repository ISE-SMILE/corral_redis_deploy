# Corral++ Redis Deployment Plugin

This is the Redis deployment plugin for [corral++](https://github.com/ISE-SMILE/corral).

`corral_redis_deploy [mode] (options...)`

## Support
Currently, supported are `docker` and `kubernetes` deployments. For `docker` we use `redis:6.2.4-alpine` Image and for `kubernetes` we use the `groundhog2k/redis` chart.

### Docker
We use Docker as a default, but you can also force docker using `local` as a mode.
This setup supports now extra options.

### Kubernetes
To use Kubernetes you need to use `k8s` or `kubernetes` as a mode.
| Name | Usage | 
| --- | ---- | 
| kubernetesNamespace | Namespace to use for the deployment | 
| kubernetesStorageClass | Storage class to use for the deployment | 
| redisPort | if set the deployment uses a NodePort with this Port | 
| kubernetesMemory | Memory resources of the Redis deployment, 512MB is the default |