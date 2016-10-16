package mock

//go:generate mockgen -package instance -destination spi/instance/instance.go github.com/docker/infrakit/spi/instance Plugin
//go:generate mockgen -package instance -destination spi/flavor/flavor.go github.com/docker/infrakit/spi/flavor Plugin
//go:generate mockgen -package client -destination docker/docker/client/api.go github.com/docker/docker/client APIClient
//go:generate mockgen -package group -destination plugin/group/group.go github.com/docker/infrakit/plugin/group Scaled
