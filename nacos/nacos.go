package nacos

import (
	"encoding/base64"
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
	Username  string // 用户名（可选）
	Password  string // 密码（可选）
}

type NacosClient struct {
	cfg          Config
	nameClient   naming_client.INamingClient
	configClient config_client.IConfigClient
	coder        Coder
}

type Coder interface {
	Encode(data string) string
	Decode(data string) (string, error)
}

var DefaultCoder = &base64Encoder{}

type options struct {
	group string
	typ   string
	coder Coder
}

type option func(*options)

func NewNacosClient(cfg Config, opts ...option) (*NacosClient, error) {
	var o = options{}
	for _, opt := range opts {
		opt(&o)
	}
	clientCofnig := constant.ClientConfig{
		NamespaceId:         cfg.Namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              cfg.LogDir,
		CacheDir:            cfg.CacheDir,
		Username:            cfg.Username,
		Password:            cfg.Password,
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
		coder:        o.coder,
	}, nil
}

func WithType(typ string) option {
	return func(o *options) {
		o.typ = typ
	}
}

func WithGroup(group string) option {
	return func(o *options) {
		o.group = group
	}
}

func WithCoder(coder Coder) option {
	return func(o *options) {
		o.coder = coder
	}
}

type Instance struct {
	ServiceName string
	IP          string
	Port        uint64
	Weight      float64
	Metadata    map[string]string
	Ephemeral   bool
}

func (n *NacosClient) RegisterInstance(i Instance, opts ...option) (bool, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          i.IP,
		Port:        i.Port,
		ServiceName: i.ServiceName,
		Weight:      i.Weight,
		GroupName:   opt.group,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   i.Ephemeral,
		Metadata:    i.Metadata,
	})
}

func (n *NacosClient) DeregisterInstance(i Instance, opts ...option) (bool, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          i.IP,
		Port:        i.Port,
		ServiceName: i.ServiceName,
		GroupName:   opt.group,
	})
}

func (n *NacosClient) BatchRegisterInstance(instances []Instance, opts ...option) (bool, error) {
	opt := n.getOpt(opts...)

	if len(instances) == 0 {
		return false, errors.New("instances is empty")
	}

	param := vo.BatchRegisterInstanceParam{
		ServiceName: instances[0].ServiceName,
		GroupName:   opt.group,
	}
	param.Instances = make([]vo.RegisterInstanceParam, 0, len(instances))
	for _, i := range instances {
		param.Instances = append(param.Instances, vo.RegisterInstanceParam{
			Ip:          i.IP,
			Port:        i.Port,
			ServiceName: i.ServiceName,
			Weight:      i.Weight,
			GroupName:   opt.group,
			Enable:      true,
		})
	}

	return n.nameClient.BatchRegisterInstance(param)
}

func (n *NacosClient) GetService(serviceName string, opts ...option) (model.Service, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.GetService(vo.GetServiceParam{
		ServiceName: serviceName,
		GroupName:   opt.group,
	})
}

func (n *NacosClient) SelectAllInstances(serviceName string, opts ...option) ([]model.Instance, error) {
	opt := n.getOpt(opts...)

	return n.nameClient.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: serviceName,
		GroupName:   opt.group,
	})
}

func (n *NacosClient) SelectInstances(serviceName string, opts ...option) ([]model.Instance, error) {
	opt := n.getOpt(opts...)
	instances, err := n.nameClient.SelectInstances(vo.SelectInstancesParam{
		ServiceName: serviceName,
		GroupName:   opt.group,
		HealthyOnly: true,
	})

	return instances, err
}

func (n *NacosClient) SelectOneHealthyInstance(serviceName string, opts ...option) (*model.Instance, error) {
	opt := n.getOpt(opts...)
	return n.nameClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
		ServiceName: serviceName,
		GroupName:   opt.group,
	})
}

const NumberRetries = 3

func (n *NacosClient) SetConfig(dataId string, content string, opts ...option) error {
	o := n.getOpt(opts...)

	var err error

	if n.coder != nil {
		content = n.coder.Encode(content)
	}

	retries := NumberRetries
	for published := false; !published && retries > 0; retries-- {
		published, err = n.configClient.PublishConfig(vo.ConfigParam{
			DataId:  dataId,
			Group:   o.group,
			Type:    o.typ,
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
func (n *NacosClient) GetConfig(dataId string, cfg any, opts ...option) error {
	c, err := n.GetConfigString(dataId, opts...)
	if err != nil {
		return err
	}

	if c == "" {
		return ErrConfigNotFound
	}

	if n.coder != nil {
		c, err = n.coder.Decode(c)
		if err != nil {
			return err
		}
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
		Group:  o.group,
	})

	return c, err
}

func (n *NacosClient) ExistConfig(dataId string, opts ...option) (bool, error) {
	o := n.getOpt(opts...)
	c, err := n.configClient.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  o.group,
	})
	if err != nil {
		return false, err
	}

	return c != "", nil
}

func (n *NacosClient) DeleteConfig(dataId string, opts ...option) error {
	o := n.getOpt(opts...)
	_, err := n.configClient.DeleteConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  o.group,
	})

	return err
}
func (n *NacosClient) ListenConfig(dataId string, changed func(namespace, group, dataId, data string), opts ...option) error {
	o := n.getOpt(opts...)

	return n.configClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  o.group,
		OnChange: func(namespace, group, dataId, data string) {
			changed(namespace, group, dataId, data)
		},
	})
}

func (n *NacosClient) CancelListenConfig(dataId string, opts ...option) error {
	o := n.getOpt(opts...)
	return n.configClient.CancelListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  o.group,
	})
}

func (n *NacosClient) getOpt(opts ...option) *options {
	o := &options{
		group: n.cfg.Group,
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

type base64Encoder struct{}

func (base64Encoder) Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func (base64Encoder) Decode(data string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}
