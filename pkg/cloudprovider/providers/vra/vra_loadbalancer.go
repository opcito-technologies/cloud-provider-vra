

package vra

import (
	"io/ioutil"
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"encoding/json"
	"net/http"
	"k8s.io/klog"
	"k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	"strconv"
)

const (

	ServiceAnnotationProjectID    = "loadbalancer.vra.org/project-id"

	// ServiceAnnotationLoadBalancerInternal is the annotation used on the service
	// to indicate that we want an internal loadbalancer service.
	// If the value of ServiceAnnotationLoadBalancerInternal is false, it indicates that we want an external loadbalancer service. Default to false.
	ServiceAnnotationLoadBalancerInternetFacing = "service.beta.kubernetes.io/vra-internet-facing-load-balancer"
	ServiceAnnotationLoadBalancerNicDeviceIndex = "service.beta.kubernetes.io/vra-load-balancer-nics-index"
	ServiceAnnotationLoadBalancerNetworkId = "service.beta.kubernetes.io/vra-load-balancer-network-id"
	ServiceAnnotationLoadBalancerAddress = "service.beta.kubernetes.io/vra-load-balancer-address"
	// ServiceAnnotationLoadBalancerSecurityGroups = "service.beta.kubernetes.io/vra-load-balancer-security-group"
	ServiceAnnotationLoadBalancerTargetLinks = "service.beta.kubernetes.io/vra-load-balancer-target-links"

	ServiceAnnotationLoadBalancerHealthyThreshold = "service.beta.kubernetes.io/vra-load-balancer-healthy-threshold"
	ServiceAnnotationLoadBalancerUnHealthyThreshold = "service.beta.kubernetes.io/vra-load-balancer-unhealthy-threshold"
	ServiceAnnotationLoadBalancerTimeOut = "service.beta.kubernetes.io/vra-load-balancer-timeout"
	ServiceAnnotationLoadBalancerIntervalSeconds = "service.beta.kubernetes.io/vra-load-balancer-interval-seconds"
	ServiceAnnotationLoadBalancerUrlPath = "service.beta.kubernetes.io/vra-load-balancer-urlpath"

)


type  VraTokenResp struct {
	TokenVal string `json:"token"`
}


type LbaasV2 struct {
	LoadBalancer
}

type empty struct{}


func getSecurityGroupName(service *v1.Service) string {
	securityGroupName := fmt.Sprintf("lb-sg-%s-%s-%s", service.UID, service.Namespace, service.Name)
	if len(securityGroupName) > 255 {
		securityGroupName = securityGroupName[:255]
	}
	return securityGroupName
}

// GetLoadBalancer returns whether the specified load balancer exists and its status
func (lbaas *LbaasV2) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {
	
	status := &v1.LoadBalancerStatus{}

	return status, true, nil
}


// todo
func (lbaas *LbaasV2)  UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {

	return nil
}

// todo
func (lbaas *LbaasV2) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	return nil
}

// GetLoadBalancerName is an implementation of LoadBalancer.GetLoadBalancerName.
func (lbaas *LbaasV2) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	// TODO: replace DefaultLoadBalancerName to generate more meaningful loadbalancer names.
	return cloudprovider.DefaultLoadBalancerName(service)
}

func nodeAddressForLB(node *v1.Node) (string, error) {
	addrs := node.Status.Addresses
	if len(addrs) == 0 {
		return "", ErrNoAddressFound
	}

	allowedAddrTypes := []v1.NodeAddressType{v1.NodeInternalIP, v1.NodeExternalIP}

	for _, allowedAddrType := range allowedAddrTypes {
		for _, addr := range addrs {
			if addr.Type == allowedAddrType {
				return addr.Address, nil
			}
		}
	}

	return "", ErrNoAddressFound
}


//getStringFromServiceAnnotation searches a given v1.Service for a specific annotationKey and either returns the annotation's value or a specified defaultSetting
func getStringFromServiceAnnotation(service *v1.Service, annotationKey string, defaultSetting string) string {
	klog.V(4).Infof("getStringFromServiceAnnotation(%v, %v, %v)", service, annotationKey, defaultSetting)
	if annotationValue, ok := service.Annotations[annotationKey]; ok {
		//if there is an annotation for this setting, set the "setting" var to it
		// annotationValue can be empty, it is working as designed
		// it makes possible for instance provisioning loadbalancer without floatingip
		klog.V(4).Infof("Found a Service Annotation: %v = %v", annotationKey, annotationValue)
		return annotationValue
	}
	//if there is no annotation, set "settings" var to the value from cloud config
	klog.V(4).Infof("Could not find a Service Annotation; falling back on cloud-config setting: %v = %v", annotationKey, defaultSetting)
	return defaultSetting
}

//getBoolFromServiceAnnotation searches a given v1.Service for a specific annotationKey and either returns the annotation's value or a specified defaultSetting
func getBoolFromServiceAnnotation(service *v1.Service, annotationKey string, defaultSetting bool) (bool, error) {
	klog.V(4).Infof("getBoolFromServiceAnnotation(%v, %v, %v)", service, annotationKey, defaultSetting)
	if annotationValue, ok := service.Annotations[annotationKey]; ok {
		returnValue := false
		switch annotationValue {
		case "true":
			returnValue = true
		case "false":
			returnValue = false
		default:
			return returnValue, fmt.Errorf("unknown %s annotation: %v, specify \"true\" or \"false\" ", annotationKey, annotationValue)
		}

		klog.V(4).Infof("Found a Service Annotation: %v = %v", annotationKey, returnValue)
		return returnValue, nil
	}
	klog.V(4).Infof("Could not find a Service Annotation; falling back to default setting: %v = %v", annotationKey, defaultSetting)
	return defaultSetting, nil
}


// isSecurityGroupNotFound return true while 'err' is object of gophercloud.ErrResourceNotFound
func isSecurityGroupNotFound(err error) bool {
	errType := reflect.TypeOf(err).String()
	errTypeSlice := strings.Split(errType, ".")
	errTypeValue := ""
	if len(errTypeSlice) != 0 {
		errTypeValue = errTypeSlice[len(errTypeSlice)-1]
	}
	if errTypeValue == "ErrResourceNotFound" {
		return true
	}

	return false
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one.
func (lbaas *LbaasV2) EnsureLoadBalancer(ctx context.Context, clusterName string, apiService *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {

	klog.V(4).Infof("EnsureLoadBalancer(%v, %v, %v, %v, %v, %v, %v)", clusterName, apiService.Namespace, apiService.Name, apiService.Spec.LoadBalancerIP, apiService.Spec.Ports, nodes, apiService.Annotations)

	if len(nodes) == 0 {
		return nil, fmt.Errorf("there are no available nodes for LoadBalancer service %s/%s", apiService.Namespace, apiService.Name)
	}

	lbname := lbaas.GetLoadBalancerName(ctx, clusterName, apiService)

    var internalAnnotation bool
    internal := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerInternetFacing, "true")

	switch internal {
	case "true":
		klog.V(4).Infof("Ensure an internal loadbalancer service.")
		internalAnnotation = true
	case "false":
		internalAnnotation = false
	default:
		internalAnnotation = true
	}

    lbaas.opts.InternetFacing = internalAnnotation


	nicDeviceIndex := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerNicDeviceIndex, "1")

	networkId := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerNetworkId, "")

	nicaddress := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerAddress, "")

	lbhlchkhealthythreshold := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerHealthyThreshold, "2")
	lbhlchkunhealthythreshold := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerUnHealthyThreshold, "2")
	lbhlchktimeout := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerTimeOut, "5")
	lbhlchkinterval := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerIntervalSeconds, "60")
	lbhlchkurlpath := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerUrlPath, "/index.html")

	// lbsggroup := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerSecurityGroups, "")


    var lbaddress[] string

    lbaddress = append(lbaddress, nicaddress)

    var lbniccustconf LBNicCustomConfigOpts
	var lbnic LBNicOpts
	var lbhlchk LBHealthCheckOpts
	var lbroute LBRouteOpts

	var vratoken VraTokenResp
	var lbreq LoadBalancerOpts

	lbniccustconf.AwaitIp = true
	lbnic.DeviceIndex, _ = strconv.Atoi(nicDeviceIndex)
    lbnic.Addresses = lbaddress
    lbnic.Name = lbname
    lbnic.NetworkId = networkId
    lbnic.CustomProperties = lbniccustconf

    lbhlchk.HealthyThreshold, _ = strconv.Atoi(lbhlchkhealthythreshold)
	lbhlchk.UnhealthyThreshold, _ = strconv.Atoi(lbhlchkunhealthythreshold)
	lbhlchk.TimeoutSeconds, _ = strconv.Atoi(lbhlchktimeout)
	lbhlchk.IntervalSeconds, _ = strconv.Atoi(lbhlchkinterval)
	lbhlchk.URLPath = lbhlchkurlpath

	ports := apiService.Spec.Ports

	if len(ports) == 0 {
		return nil, fmt.Errorf("no ports provided to vra load balancer")
	}

	for _, port := range ports {
		lbhlchk.Protocol = string(port.Protocol)
		lbhlchk.Port = string(port.Port)
		lbroute.Protocol  = string(port.Protocol)
		lbroute.Port = string(port.Port)
		lbroute.MemberPort = string(port.Port)
		lbroute.MemberProtocol = string(port.Protocol)
	}

	
	lbroute.HealthCheckConfiguration = lbhlchk

	lbreq.Routes = [] LBRouteOpts { lbroute }
	lbreq.Nics = [] LBNicOpts { lbnic }

	lbreq.ProjectID =   getStringFromServiceAnnotation(apiService, ServiceAnnotationProjectID, "df87d5e2-ac4e-4b38-8d6f-a6260dc63e95")
	
	lbreq.Description = "vra lb"
	lbreq.Name = lbname
	lbreq.InternetFacing = true

	targetLinks := getStringFromServiceAnnotation(apiService, ServiceAnnotationLoadBalancerTargetLinks, "")
	var target = [] string {targetLinks}

	lbreq.TargetLinks =  target  

	logindata := map[string]string{"refreshToken": lbaas.authToken}
	loginreq, _ := json.Marshal(logindata)

	lbjsonStr, _ := json.Marshal(lbreq)

	response, err := http.Post(lbaas.apiHost+"/iaas/login", "application/json", bytes.NewBuffer(loginreq))

	if err != nil {
		fmt.Printf("The login failed")
	} else {
		tokendata, _ := ioutil.ReadAll(response.Body)
		err = json.Unmarshal(tokendata, &vratoken)
	}

	// for creating the lb
	lbrequest, _ := http.NewRequest("POST", lbaas.apiHost+"/iaas/api/load-balancers", bytes.NewBuffer(lbjsonStr))
	token := vratoken.TokenVal
	lbrequest.Header.Add("Accept", "application/json")
	lbrequest.Header.Add("Authorization", "Bearer " + token)

	lbresp, err := http.DefaultClient.Do(lbrequest)

	if err != nil {
		fmt.Printf("the http req error")
	} else {
		lbrespdata, _ := ioutil.ReadAll(lbresp.Body)
		fmt.Println(lbrespdata)
	}
	status := &v1.LoadBalancerStatus{}

	return status, nil
}
