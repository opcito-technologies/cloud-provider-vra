
package vra

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"
	"gopkg.in/gcfg.v1"
	"errors"

	// "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
)

const (
	// ProviderName is the name of the vra provider
	ProviderName = "vra"
	defaultTimeOut   = 60 * time.Second
	availabilityZone = "availability_zone"
)

var ErrNoAddressFound = errors.New("no address found for host")

type MyDuration struct {
	time.Duration
}

// UnmarshalText is used to convert from text to Duration
func (d *MyDuration) UnmarshalText(text []byte) error {
	res, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = res
	return nil
}

type LBNicCustomConfigOpts struct {
	AwaitIp bool `json:"awaitIp"`
}

type LBNicOpts struct {
	Name string `json:"name"`
	Description string `json:"description"`
	DeviceIndex int `json:"deviceIndex"`
	NetworkId string `json:"networkId"`
	Addresses [] string `json:"addresses"`
	CustomProperties LBNicCustomConfigOpts `json:"customProperties"`
	SecurityGroupIds [] string `json:"securityGroupIds"`
}

type LBHealthCheckOpts struct {
	Protocol           string `json:"protocol"`
	Port               string `json:"port"`
	URLPath            string `json:"urlPath"`
	IntervalSeconds    int    `json:"intervalSeconds"`
	TimeoutSeconds     int    `json:"timeoutSeconds"`
	UnhealthyThreshold int    `json:"unhealthyThreshold"`
	HealthyThreshold   int    `json:"healthyThreshold"`
}

type LBRouteOpts struct {
	Protocol                 string `json:"protocol"`
	Port                     string `json:"port"`
	MemberProtocol           string `json:"memberProtocol"`
	MemberPort               string `json:"memberPort"`
	HealthCheckConfiguration LBHealthCheckOpts `json:"healthCheckConfiguration"`
}

// LoadBalancerOpts have options for vra api
type LoadBalancerOpts struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Routes      [] LBRouteOpts `json:"routes"`
	Nics 		[] LBNicOpts `json:"nics"`
	ProjectID   string `json:"projectId"`
	InternetFacing bool `json:"internetFacing"`
	TargetLinks [] string `json:"targetLinks"`
}


// LoadBalancer is used for creating and maintaining load balancers
type LoadBalancer struct {
	authToken string
    apiHost string
	opts    LoadBalancerOpts
}


// RouterOpts have options for vra api
type RouterOpts struct {
	RouterID string `json:"router-id"` // required
}

// BlockStorageOpts have options for vra api
type BlockStorageOpts struct {
	BSVersion       string `json:"bs-version"`
	TrustDevicePath bool   `json:"trust-device-path"` // See Issue #33128
	IgnoreVolumeAZ  bool   `json:"ignore-volume-az"`
}

// vra is an implementation of cloud provider Interface for vra.
type Vra struct {
	authToken  string
	apiHost string
	lbOpts         LoadBalancerOpts
	bsOpts         BlockStorageOpts
	routeOpts      RouterOpts
	// InstanceID of the server where this vra object is instantiated.
	localInstanceID string
}

// Config is used to read and store information from the cloud configuration file
type Config struct {
	Global struct {
		APIToken string `gcfg:"api-token"`
		APIHost string `gcfg:"api-host"`
		CloudsFile string `gcfg:"clouds-file,omitempty"`
	}
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
	Route        RouterOpts
}


func init() {
	klog.Warningf("**** inside vra init before RegisterMetrics *******")
	RegisterMetrics()
	klog.Warningf("**** inside vra init after RegisterMetrics *******")
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := ReadConfig(config)
		if err != nil {
			return nil, err
		}
		cloud, err := NewVra(cfg)
		if err != nil {
			klog.V(1).Infof("New vra client created failed with config")
		}
		klog.Warningf("**** inside vra init binary ******* %q", err)
		return cloud, err
	})
	klog.Warningf("**** inside vra  init end of func *******")
}


// configFromEnv allows setting up credentials etc using the
func configFromEnv() (cfg Config, ok bool) {
	cfg.Global.APIHost = os.Getenv("VRA_API_HOST")
	cfg.Global.APIToken = os.Getenv("API_TOKEN")

	ok = cfg.Global.APIHost != "" &&
		cfg.Global.APIToken != "" 
	return
}


// ReadConfig reads values from environment variables and the cloud.conf, prioritizing cloud-config
func ReadConfig(config io.Reader) (Config, error) {
	if config == nil {
		return Config{}, fmt.Errorf("no vra cloud provider config file given")
	}

	cfg, _ := configFromEnv()

	err := gcfg.FatalOnly(gcfg.ReadInto(&cfg, config))

	return cfg, err
}


// caller is a tiny helper for conditional unwind logic
type caller bool

func newCaller() caller   { return caller(true) }
func (c *caller) disarm() { *c = false }

func (c *caller) call(f func()) {
	if *c {
		f()
	}
}

// check opts for vra
func checkVraOpts(vraOpts *Vra) error {

	return nil
}

// NewVra creates a new new instance of the vra struct from a config struct
func NewVra(cfg Config) (*Vra, error) {

	vr := Vra{
		apiHost:		cfg.Global.APIHost,
		authToken:		cfg.Global.APIToken,
		lbOpts:         cfg.LoadBalancer,
		bsOpts:         cfg.BlockStorage,
		routeOpts:      cfg.Route,
	}

	err := checkVraOpts(&vr)
	if err != nil {
		return nil, err
	}

	return &vr, nil	
}

// Initialize passes a Kubernetes clientBuilder interface to the cloud provider
func (vr *Vra) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	
}

// mapNodeNameToServerName maps a k8s NodeName to an VRA Server Name
// This is a simple string cast.
func mapNodeNameToServerName(nodeName types.NodeName) string {
	return string(nodeName)
}


// Clusters is a no-op
func (vr *Vra) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (vr *Vra) ProviderName() string {
	return ProviderName
}

// ScrubDNS filters DNS settings for pods.
func (vr *Vra) ScrubDNS(nameServers, searches []string) ([]string, []string) {
	return nameServers, searches
}

// HasClusterID returns true if the cluster has a clusterID
func (vr *Vra) HasClusterID() bool {
	return true
}

// LoadBalancer initializes object
func (vr *Vra) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	klog.V(4).Info("vra.LoadBalancer() called")

	if reflect.DeepEqual(vr.lbOpts, LoadBalancerOpts{}) {
		klog.V(4).Info("LoadBalancer section is empty/not defined in cloud-config")
		return nil, false
	}

	return &LbaasV2{LoadBalancer{vr.authToken, vr.apiHost, vr.lbOpts}}, true
}


// Zones indicates that we support zones
func (vr *Vra) Zones() (cloudprovider.Zones, bool) {
	klog.V(1).Info("Claiming to support Zones")
	return vr, true
}


// GetZone returns the current zone
func (vr *Vra) GetZone(ctx context.Context) (cloudprovider.Zone, error) {

	klog.V(4).Infof("Claiming to support GetZone")
	return cloudprovider.Zone{
		FailureDomain: availabilityZone,
		Region:        "",
	}, nil
}

// GetZoneByProviderID implements Zones.GetZoneByProviderID
// This is particularly useful in external cloud providers where the kubelet
// does not initialize node data.
func (vr *Vra) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {

    klog.V(4).Infof("Claiming to support GetZoneByProviderID")
	return cloudprovider.Zone{
		FailureDomain: availabilityZone,
		Region:        "",
	}, nil
}


// GetZoneByNodeName implements Zones.GetZoneByNodeName
// This is particularly useful in external cloud providers where the kubelet
// does not initialize node data.
func (vr *Vra) GetZoneByNodeName(ctx context.Context, nodeName types.NodeName) (cloudprovider.Zone, error) {
	klog.V(4).Infof("Claiming to support GetZoneByNodeName")
	return cloudprovider.Zone{
		FailureDomain: availabilityZone,
		Region:        "",
	}, nil
}

// Routes initializes routes support
func (vr *Vra) Routes() (cloudprovider.Routes, bool) {
	klog.V(4).Info("vra.Routes() called")
	klog.V(1).Info("Claiming to support Routes")
	return nil, true
}
