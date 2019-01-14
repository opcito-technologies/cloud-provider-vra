
package vra

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/gcfg.v1"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	netutil "k8s.io/apimachinery/pkg/util/net"
	certutil "k8s.io/client-go/util/cert"
	cloudprovider "k8s.io/cloud-provider"
	v1helper "k8s.io/cloud-provider-vra/pkg/apis/core/v1/helper"
	"k8s.io/klog"
)

const (
	// ProviderName is the name of the vra provider
	ProviderName = "vra"
	defaultTimeOut   = 60 * time.Second
)

// ErrNotFound is used to inform that the object is missing
var ErrNotFound = errors.New("failed to find object")

// ErrMultipleResults is used when we unexpectedly get back multiple results
var ErrMultipleResults = errors.New("multiple results where only one expected")

// ErrNoAddressFound is used when we cannot find an ip address for the host
var ErrNoAddressFound = errors.New("no address found for host")

// ErrIPv6SupportDisabled is used when one tries to use IPv6 Addresses when
// IPv6 support is disabled by config
var ErrIPv6SupportDisabled = errors.New("IPv6 support is disabled")

// MyDuration is the encoding.TextUnmarshaler interface for time.Duration
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

// LoadBalancer is used for creating and maintaining load balancers
type LoadBalancer struct {
	network *gophercloud.ServiceClient
	compute *gophercloud.ServiceClient
	lb      *gophercloud.ServiceClient
	opts    LoadBalancerOpts
}

// LoadBalancerOpts have the 
type LoadBalancerOpts struct {
	LBVersion            string     ``
}


// NetworkingOpts is used for networking settings
type NetworkingOpts struct {
	IPv6SupportDisabled bool   ``
	PublicNetworkName   string ``
}


// RouterOpts is used for 
type RouterOpts struct {
	RouterID string `` // required
}

// MetadataOpts is used for configuring how to talk to metadata service or config drive
type MetadataOpts struct {
	SearchOrder    string     ``
	RequestTimeout MyDuration ``
}

type ServerAttributesExt struct {
	servers.Server
	availabilityzones.ServerAvailabilityZoneExt
}


// vra is an implementation of cloud provider Interface for vra.
type vra struct {
	provider       *gophercloud.ProviderClient
	region         string
	lbOpts         LoadBalancerOpts
	bsOpts         BlockStorageOpts
	routeOpts      RouterOpts
	metadataOpts   MetadataOpts
	networkingOpts NetworkingOpts
	// InstanceID of the server where this vra object is instantiated.
	localInstanceID string
}

// Config is used to read and store information from the cloud configuration file
type Config struct {
	Global struct {
		Region     string
	}
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
	Route        RouterOpts
	Metadata     MetadataOpts
	Networking   NetworkingOpts
}


func init() {
	RegisterMetrics()

	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := ReadConfig(config)
		if err != nil {
			return nil, err
		}
		return NewVra(cfg)
	})
}

// ReadConfig reads values from environment variables and the cloud.conf, prioritizing cloud-config
func ReadConfig(config io.Reader) (Config, error) {
	if config == nil {
		return Config{}, fmt.Errorf("no vra cloud provider config file given")
	}

	cfg, _ := configFromEnv()

	// Set default values for config params
	cfg.BlockStorage.BSVersion = "auto"
	cfg.BlockStorage.TrustDevicePath = false
	cfg.BlockStorage.IgnoreVolumeAZ = false
	cfg.Metadata.SearchOrder = fmt.Sprintf("%s,%s", configDriveID, metadataID)
	cfg.Networking.IPv6SupportDisabled = false
	cfg.Networking.PublicNetworkName = "public"

	err := gcfg.FatalOnly(gcfg.ReadInto(&cfg, config))
	if cfg.Global.UseClouds {
		if cfg.Global.CloudsFile != "" {
			os.Setenv("OS_CLIENT_CONFIG_FILE", cfg.Global.CloudsFile)
		}
		err = ReadClouds(&cfg)
		if err != nil {
			return Config{}, err
		}
	}
	return cfg, err
}


func replaceEmpty(a string, b string) string {
	if a == "" {
		return b
	}
	return a
}

// ReadClouds reads Reads clouds.yaml to generate a Config
// Allows the cloud-config to have priority
func ReadClouds(cfg *Config) error {

	co := new(clientconfig.ClientOpts)
	cloud, err := clientconfig.GetCloudFromYAML(co)
	if err != nil && err.Error() != "unable to load clouds.yaml: no clouds.yaml file found" {
		return err
	}

	cfg.Global.AuthURL = replaceEmpty(cfg.Global.AuthURL, cloud.AuthInfo.AuthURL)
	return nil
}

// caller is a tiny helper for conditional unwind logic
type caller bool


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
	lbOpts := vraOpts.lbOpts


	emptyDuration := MyDuration {}
	if lbOpts.CreateMonitor {
		if lbOpts.MonitorDelay == emptyDuration {
			return fmt.Errorf("monitor-delay not set in cloud provider config")
		}
		if lbOpts.MonitorTimeout == emptyDuration {
			return fmt.Errorf("monitor-timeout not set in cloud provider config")
		}
		if lbOpts.MonitorMaxRetries == uint(0) {
			return fmt.Errorf("monitor-max-retries not set in cloud provider config")
		}
	}
	
}

// NewVra creates a new new instance of the vra struct from a config struct
func NewVra(cfg Config) (*Vra, error) {
	provider, err := vra.NewClient(cfg.Global.AuthURL)

	if err != nil {
	   return nil, err
	}


	vr := Vra{
		provider:       provider,
		region:         cfg.Global.Region,
		lbOpts:         cfg.LoadBalancer,
		bsOpts:         cfg.BlockStorage,
		routeOpts:      cfg.Route,
		metadataOpts:   cfg.Metadata,
		networkingOpts: cfg.Networking,
	}

	err = checkVraOpts(&vr)
	if err != nil {
		return nil, err
	}

	return &vr, nil	
}

// Initialize passes a Kubernetes clientBuilder interface to the cloud provider
func (os *Vra) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
}

// mapNodeNameToServerName maps a k8s NodeName to an VRA Server Name
// This is a simple string cast.
func mapNodeNameToServerName(nodeName types.NodeName) string {
	return string(nodeName)
}

// GetNodeNameByID maps instanceid to types.NodeName
func (vr *Vra) GetNodeNameByID(instanceID string) (types.NodeName, error) {
	client, err := vr.NewComputeV2()
	var nodeName types.NodeName
	if err != nil {
		return nodeName, err
	}

	// todo - server, err := servers.Get(client, instanceID).Extract()
	if err != nil {
		return nodeName, err
	}
	nodeName = mapServerToNodeName(server)
	return nodeName, nil
}


// mapServerToNodeName maps an Vra Server to a k8s NodeName
func mapServerToNodeName(server *servers.Server) types.NodeName {
	// Node names are always lowercase, and (at least)
	// routecontroller does case-sensitive string comparisons
	// assuming this
	return types.NodeName(strings.ToLower(server.Name))
}


// todo
func foreachServer(client *gophercloud.ServiceClient, opts servers.ListOptsBuilder, handler func(*servers.Server) (bool, error)) error {
}

// todo
func getServerByName(client *gophercloud.ServiceClient, name types.NodeName) (*ServerAttributesExt, error) {
	opts := servers.ListOpts{
		Name: fmt.Sprintf("^%s$", regexp.QuoteMeta(mapNodeNameToServerName(name))),
	}

}


// todo
func nodeAddresses(srv *servers.Server, networkingOpts NetworkingOpts) ([]v1.NodeAddress, error) {
	addrs := []v1.NodeAddress{}

	type Address struct {
		IPType string `mapstructure:"OS-EXT-IPS:type"`
		Addr   string
	}

	return addrs, nil
}


// todo
func getAddressesByName(client *gophercloud.ServiceClient, name types.NodeName, networkingOpts NetworkingOpts) ([]v1.NodeAddress, error) {
	srv, err := getServerByName(client, name)
	if err != nil {
		return nil, err
	}

	return nodeAddresses(&srv.Server, networkingOpts)
}


// todo
func getAddressByName(client *gophercloud.ServiceClient, name types.NodeName, needIPv6 bool, networkingOpts NetworkingOpts) (string, error) {
	if needIPv6 && networkingOpts.IPv6SupportDisabled {
		return "", ErrIPv6SupportDisabled
	}

}

// todo

// getAttachedInterfacesByID returns the node interfaces of the specified instance.
func getAttachedInterfacesByID(client *gophercloud.ServiceClient, serviceID string) ([]attachinterfaces.Interface, error) {
	var interfaces []attachinterfaces.Interface

	return interfaces, nil
}

// todo

// Clusters is a no-op
func (os *Vra) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID.
func (os *Vra) ProviderName() string {
	return ProviderName
}

// ScrubDNS filters DNS settings for pods.
func (os *Vra) ScrubDNS(nameServers, searches []string) ([]string, []string) {
	return nameServers, searches
}

// HasClusterID returns true if the cluster has a clusterID
func (os *Vra) HasClusterID() bool {
	return true
}

// LoadBalancer initializes a LbaasV2 object
func (os *Vra) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	klog.V(4).Info("vra.LoadBalancer() called")

	if reflect.DeepEqual(os.lbOpts, LoadBalancerOpts{}) {
		klog.V(4).Info("LoadBalancer section is empty/not defined in cloud-config")
		return nil, false
	}

	// network, err := os.NewNetworkV2()
	if err != nil {
		return nil, false
	}

	// compute, err := os.NewComputeV2()
	if err != nil {
		return nil, false
	}

	// lb, err := os.NewLoadBalancerV2()
	if err != nil {
		return nil, false
	}
		
	// lbVersion := os.lbOpts.LBVersion
	if lbVersion != "" && lbVersion != "v2" {
		klog.Warningf("Config error: currently only support LBaaS v2, unrecognised lb-version \"%v\"", lbVersion)
		return nil, false
	}

	// klog.V(1).Info("Claiming to support LoadBalancer")

	// return &LbaasV2{LoadBalancer{network, compute, lb, os.lbOpts}}, true
}

func isNotFound(err error) bool {
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		return true
	}

	if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
		if errCode.Actual == http.StatusNotFound {
			return true
		}
	}

	return false
}

// Zones indicates that we support zones
func (vr *Vra) Zones() (cloudprovider.Zones, bool) {
	klog.V(1).Info("Claiming to support Zones")
	return vr, true
}

// GetZone returns the current zone
func (vr *Vra) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	md, err := getMetadata(os.metadataOpts.SearchOrder)
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	zone := cloudprovider.Zone{
		FailureDomain: md.AvailabilityZone,
		Region:        os.region,
	}
	klog.V(4).Infof("Current zone is %v", zone)
	return zone, nil
}


// GetZoneByProviderID implements Zones.GetZoneByProviderID
// This is particularly useful in external cloud providers where the kubelet
// does not initialize node data.
func (vr *Vra) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	instanceID, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	// todo -compute, err := os.NewComputeV2()
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	var serverWithAttributesExt ServerAttributesExt
	if err := servers.Get(compute, instanceID).ExtractInto(&serverWithAttributesExt); err != nil {
		return cloudprovider.Zone{}, err
	}

	zone := cloudprovider.Zone{
		FailureDomain: serverWithAttributesExt.AvailabilityZone,
		Region:        os.region,
	}
	klog.V(4).Infof("The instance %s in zone %v", serverWithAttributesExt.Name, zone)
	return zone, nil
}



// GetZoneByNodeName implements Zones.GetZoneByNodeName
// This is particularly useful in external cloud providers where the kubelet
// does not initialize node data.
func (vr *Vra) GetZoneByNodeName(ctx context.Context, nodeName types.NodeName) (cloudprovider.Zone, error) {
	 // todo -compute, err := os.NewComputeV2()
	if err != nil {
		return cloudprovider.Zone{}, err
	}

	// todo
	srv, err := getServerByName(compute, nodeName)
	
	if err != nil {
		if err == ErrNotFound {
			return cloudprovider.Zone{}, cloudprovider.InstanceNotFound
		}
		return cloudprovider.Zone{}, err
	}

	zone := cloudprovider.Zone{
		FailureDomain: srv.AvailabilityZone,
		Region:        os.region,
	}
	klog.V(4).Infof("The instance %s in zone %v", srv.Name, zone)
	return zone, nil
}

// Routes initializes routes support
func (vr *Vra) Routes() (cloudprovider.Routes, bool) {
	klog.V(4).Info("vra.Routes() called")

	// todo - network, err := vr.NewNetworkV2()
	if err != nil {
		return nil, false
	}

	// todo - netExts, err := networkExtensions(network)
	if err != nil {
		klog.Warningf("Failed to list network extensions: %v", err)
		return nil, false
	}

	if !netExts["extraroute"] {
		klog.V(3).Info("Neutron extraroute extension not found, required for Routes support")
		return nil, false
	}

	// todo - compute, err := vr.NewComputeV2()
	if err != nil {
		return nil, false
	}

	// todo -	r, err := NewRoutes(compute, network, vr.routeOpts, vr.networkingOpts)
	if err != nil {
		klog.Warningf("Error initialising Routes support: %v", err)
		return nil, false
	}

	klog.V(1).Info("Claiming to support Routes")
	return r, true
}

func (vr *vra) volumeService(forceVersion string) (volumeService, error) {
	bsVersion := ""
	if forceVersion == "" {
		bsVersion = os.bsOpts.BSVersion
	} else {
		bsVersion = forceVersion
	}

	switch bsVersion {
	case "v1":
		// todo - sClient, err := vr.NewBlockStorageV1()
		if err != nil {
			return nil, err
		}
		klog.V(3).Info("Using Blockstorage API V1")
		return &VolumesV1{sClient, vr.bsOpts}, nil
	case "v2":
		// todo - sClient, err := os.NewBlockStorageV2()
		if err != nil {
			return nil, err
		}
		klog.V(3).Info("Using Blockstorage API V2")
		return &VolumesV2{sClient, vr.bsOpts}, nil
	case "v3":
		// todo - sClient, err := os.NewBlockStorageV3()
		if err != nil {
			return nil, err
		}
		klog.V(3).Info("Using Blockstorage API V3")
		return &VolumesV3{sClient, vr.bsOpts}, nil
	case "auto":
		
		// Choose Cinder v3 firstly, if kubernetes can't initialize cinder v3 client, try to initialize cinder v2 client.
		// If kubernetes can't initialize cinder v2 client, try to initialize cinder v1 client.
		// Return appropriate message when kubernetes can't initialize them.
		if sClient, err := vr.NewBlockStorageV3(); err == nil {
			klog.V(3).Info("Using Blockstorage API V3")
			return &VolumesV3{sClient, vr.bsOpts}, nil
		}

		if sClient, err := vr.NewBlockStorageV2(); err == nil {
			klog.V(3).Info("Using Blockstorage API V2")
			return &VolumesV2{sClient, vr.bsOpts}, nil
		}

		if sClient, err := vr.NewBlockStorageV1(); err == nil {
			klog.V(3).Info("Using Blockstorage API V1")
			return &VolumesV1{sClient, vr.bsOpts}, nil
		}

		errTxt := "BlockStorage API version autodetection failed. " +
			"Please set it explicitly in cloud.conf in section [BlockStorage] with key `bs-version`"
		return nil, errors.New(errTxt)
	default:
		errTxt := fmt.Sprintf("Config error: unrecognised bs-version \"%v\"", os.bsOpts.BSVersion)
		return nil, errors.New(errTxt)
	}
}

func checkMetadataSearchOrder(order string) error {
	if order == "" {
		return errors.New("invalid value in section [Metadata] with key `search-order`. Value cannot be empty")
	}

	elements := strings.Split(order, ",")
	if len(elements) > 2 {
		return errors.New("invalid value in section [Metadata] with key `search-order`. Value cannot contain more than 2 elements")
	}

	for _, id := range elements {
		id = strings.TrimSpace(id)
		switch id {
		case configDriveID:
		case metadataID:
		default:
			return fmt.Errorf("invalid element %q found in section [Metadata] with key `search-order`."+
				"Supported elements include %q and %q", id, configDriveID, metadataID)
		}
	}

	return nil
}
