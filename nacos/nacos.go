package nacos

import (
	"encoding/json"
	"errors"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

var (
	ErrConfigNotFound = errors.New("config not found")
	ErrPublishFailed  = errors.New("publish failed")
)

type Config struct {
	Host      string // nacos 地址
	Port      int    // nacos 端口
	GrpcPort  int    // nacos grpc 端口
	Namespace string // nacos 命名空间
	Group     string // nacos 配置分组
	CacheDir  string
	LogDir    string
}

type NacosClient struct {
	cfg          Config
	nameClient   naming_client.INamingClient
	configClient config_client.IConfigClient
}

type options struct {
	Group string
	Type  string
}

type option func(*options)

func NewNacosClient(cfg Config) (*NacosClient, error) {
	clientCofnig := constant.ClientConfig{
		NamespaceId:         cfg.Namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              cfg.LogDir,
		CacheDir:            cfg.CacheDir,
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:   cfg.Host,
			Port:     uint64(cfg.Port),
			GrpcPort: uint64(cfg.GrpcPort),
		},
	}

	nameClient, err := clients.CreateNamingClient(map[string]any{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientCofnig,
	})
	if err != nil {
		return nil, err
	}

	configClient, err := clients.CreateConfigClient(map[string]any{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientCofnig,
	})
	if err != nil {
		return nil, err
	}

	return &NacosClient{
		cfg:          cfg,
		nameClient:   nameClient,
		configClient: configClient,
	}, nil
}

func WithType(typ string) option {
	return func(o *options) {
		o.Type = typ
	}
}

func WithGroup(group string) option {
	return func(o *options) {
		o.Group = group
	}
}

type Instance struct {
	ServiceName string
	IP          string
	Port        uint64
	Weight      float64
	Ephemeral   bool
}

func (n *NacosClient) RegisterInstance(i Instance, opts ...option) (bool, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          i.IP,
		Port:        i.Port,
		ServiceName: i.ServiceName,
		Weight:      i.Weight,
		GroupName:   opt.Group,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   i.Ephemeral,
	})
}

func (n *NacosClient) BatchRegisterInstance(instances []Instance, opts ...option) (bool, error) {
	opt := n.getOpt(opts...)

	if len(instances) == 0 {
		return false, errors.New("instances is empty")
	}

	param := vo.BatchRegisterInstanceParam{
		ServiceName: instances[0].ServiceName,
		GroupName:   opt.Group,
	}
	param.Instances = make([]vo.RegisterInstanceParam, 0, len(instances))
	for _, i := range instances {
		param.Instances = append(param.Instances, vo.RegisterInstanceParam{
			Ip:          i.IP,
			Port:        i.Port,
			ServiceName: i.ServiceName,
			Weight:      i.Weight,
			GroupName:   opt.Group,
			Enable:      true,
		})
	}

	return n.nameClient.BatchRegisterInstance(param)
}

func (n *NacosClient) DeregisterInstance(i Instance, opts ...option) (bool, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          i.IP,
		Port:        i.Port,
		ServiceName: i.ServiceName,
		GroupName:   opt.Group,
	})
}

func (n *NacosClient) GetService(serviceName string, opts ...option) (model.Service, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.GetService(vo.GetServiceParam{
		ServiceName: serviceName,
		GroupName:   opt.Group,
	})
}

func (n *NacosClient) SelectAllInstances(serviceName string, opts ...option) ([]model.Instance, error) {
	opt := n.getOpt(opts...)

	return n.nameClient.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   opt.Group,
	})
}

func (n *NacosClient) SelectInstances(serviceName string, opts ...option) ([]model.Instance, error) {
	opt := n.getOpt(opts...)
	instances, err := n.nameClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   opt.Group,
		HealthyOnly: true,
	})

	return instances, err
}

func (n *NacosClient) SelectOneHealthyInstance(serviceName string, opts ...option) (*model.Instance, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   opt.Group,
	})
}

const NumberRetries = 3

func (n *NacosClient) SetConfig(dataId string, content string, opts ...option) error {
	o := n.getOpt(opts...)

	var err error

	retries := NumberRetries
	for published := false; !published && retries > 0; retries-- {
		published, err = n.configClient.PublishConfig(vo.ConfigParam{
			DataId:  dataId,
			Group:   o.Group,
			Type:    o.Type,
			Content: content,
		})

		if err != nil {
			return err
		}

		if published {
			return nil
		}
	}

	return ErrPublishFailed
}

// 获取配置文件
func (n *NacosClient) GetConfig(cfg any, dataId string, opts ...option) error {
	c, err := n.GetConfigString(dataId, opts...)
	if err != nil {
		return err
	}

	if c == "" {
		return ErrConfigNotFound
	}

	if err := json.Unmarshal([]byte(c), cfg); err != nil {
		return err
	}

	return nil
}

func (n *NacosClient) GetConfigString(dataId string, opts ...option) (string, error) {
	o := n.getOpt(opts...)

	c, err := n.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  o.Group,
	})

	return c, err
}

func (n *NacosClient) getOpt(opts ...option) *options {
	o := &options{
		Group: n.cfg.Group,
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}
