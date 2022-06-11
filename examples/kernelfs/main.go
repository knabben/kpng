/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"k8s.io/klog/v2"
	"net"
	"sigs.k8s.io/kpng/api/localnetv1"
	"sigs.k8s.io/kpng/client"
	"sync"
)

// change this to setup
var (
	lock sync.RWMutex
)

type NoopNodeHandler struct{}

type Proxier struct {
	NoopNodeHandler

	// endpointsChanges and serviceChanges contains all changes to endpoints and
	// services that happened since policies were synced. For a single object,
	// changes are accumulated, i.e. previous is state from before all of them,
	// current is state after applying all of those.
	//endpointsChanges  *proxy.EndpointChangeTracker
	//serviceChanges    *proxy.ServiceChangeTracker

	//endPointsRefCount endPointsReferenceCountMap
	mu sync.Mutex // protects the following fields
	//serviceMap        proxy.ServiceMap
	//endpointsMap      proxy.EndpointsMap
	// endpointSlicesSynced and servicesSynced are set to true when corresponding
	// objects are synced after startup. This is used to avoid updating hns policies
	// with some partial data after kube-proxy restart.
	endpointSlicesSynced bool
	servicesSynced       bool
	isIPv6Mode           bool
	initialized          int32
	//syncRunner           *async.BoundedFrequencyRunner // governs calls to syncProxyRules
	// These are effectively const and do not need the mutex to be held.
	masqueradeAll  bool
	masqueradeMark string
	clusterCIDR    string
	hostname       string
	nodeIP         net.IP
	//recorder       events.EventRecorder

	//serviceHealthServer healthcheck.ServiceHealthServer
	//healthzServer       healthcheck.ProxierHealthUpdater

	// Since converting probabilities (floats) to strings is expensive
	// and we are using only probabilities in the format of 1/n, we are
	// precomputing some number of those and cache for future reuse.
	precomputedProbabilities []string

	hns       HostNetworkService
	network   HnsNetworkInfo
	sourceVip string
	hostMac   string
	isDSR     bool
	//supportedFeatures hcn.SupportedFeatures
	healthzPort int

	forwardHealthCheckVip bool
	rootHnsEndpointName   string
}

func Enum(p localnetv1.Protocol) uint16 {
	if p.String() == "TCP" {
		return 6
	}
	if p.String() == "UDP" {
		return 17
	}
	return 0
}

// todo(knabben) - need a mapping of services and localnetv1.services
/*func newHostNetworkService() (HostNetworkService, hcn.SupportedFeatures) {
	var hns HostNetworkService
	hns = hnsV1{}
	supportedFeatures := hcn.GetSupportedFeatures()
	if supportedFeatures.Api.V2 {
		hns = hnsV2{}
	}

	return hns, supportedFeatures
}*/

func main() {
	client.Run(Callback)
}

var proxier *Proxier

func Callback(items []*client.ServiceEndpoints) {
	masqueradeValue := 1 << uint(1)
	masqueradeMark := fmt.Sprintf("%#08x/%#08x", masqueradeValue, masqueradeValue)

	proxier = &Proxier{
		masqueradeAll:  true,
		masqueradeMark: masqueradeMark,
		clusterCIDR:    "",
		hostname:       "",
		nodeIP:         net.IP{},
		hostMac:        "",
		isDSR:          false,
		isIPv6Mode:     false,
		healthzPort:    80,
		hns:            HnsV1{},
		//network:        nil,
	}

	syncProxyRules(items)
}

func newSourceVIP(hns HostNetworkService, network string, ip string, mac string, providerAddress string) (*EndpointsInfo, error) {
	hnsEndpoint := &EndpointsInfo{
		ip:              ip,
		isLocal:         true,
		macAddress:      mac,
		providerAddress: providerAddress,

		ready:       true,
		serving:     true,
		terminating: false,
	}
	ep, err := hns.createEndpoint(hnsEndpoint, network)
	return ep, err
}

// This is where all of the hns save/restore calls happen.
// assumes proxier.mu is held
func syncProxyRules(items []*client.ServiceEndpoints) {
	lock.Lock()
	defer lock.Unlock()

	hnsNetworkName := "External" // proxier.network.name
	hns := proxier.hns

	//todo(knabben) - clean stale services

	// Query HNS for endpoints and load balancers
	queriedEndpoints, err := hns.GetAllEndpointsByNetwork(hnsNetworkName)
	if err != nil {
		klog.ErrorS(err, "Querying HNS for endpoints failed")
		return
	}
	if queriedEndpoints == nil {
		klog.V(4).InfoS("No existing endpoints found in HNS")
		queriedEndpoints = make(map[string]*(EndpointsInfo))
	}
	queriedLoadBalancers, err := hns.getAllLoadBalancers()
	if queriedLoadBalancers == nil {
		klog.V(4).InfoS("No existing load balancers found in HNS")
		queriedLoadBalancers = make(map[loadBalancerIdentifier]*(loadBalancerInfo))
	}
	if err != nil {
		klog.ErrorS(err, "Querying HNS for load balancers failed")
		return
	}

	//if strings.EqualFold(proxier.network.networkType, NETWORK_TYPE_OVERLAY) {
	proxier.sourceVip = "192.168.255.2"
	if _, ok := queriedEndpoints[proxier.sourceVip]; !ok {
		_, err = newSourceVIP(hns, hnsNetworkName, proxier.sourceVip, proxier.hostMac, "192.168.255.3")
		if err != nil {
			klog.ErrorS(err, "Source Vip endpoint creation failed")
			return
		}
	}

	fmt.Println("Syncing Policies")

	// Program HNS by adding corresponding policies for each service.
	for _, svcEndpoints := range items {
		fmt.Println(fmt.Sprintf("service -- %v", svcEndpoints.Service))
		svcInfo := svcEndpoints.Service

		svcip := svcInfo.IPs.ClusterIPs.V4[0]
		serviceVipEndpoint := queriedEndpoints[svcip]
		if serviceVipEndpoint == nil {
			klog.V(4).InfoS("No existing remote endpoint", "IP", svcip)
			hnsEndpoint := &EndpointsInfo{
				ip:              svcip,
				isLocal:         false,
				macAddress:      "00:15:5D:C9:6C:8D",
				providerAddress: "192.168.255.2", //proxier.nodeIP.String(),
			}

			newHnsEndpoint, err := hns.createEndpoint(hnsEndpoint, hnsNetworkName)
			if err != nil {
				klog.ErrorS(err, "Remote endpoint creation failed for service VIP")
				continue
			}

			//newHnsEndpoint.refCount = proxier.endPointsRefCount.getRefCount(newHnsEndpoint.hnsID)
			//*newHnsEndpoint.refCount++
			//svcInfo.remoteEndpoint = newHnsEndpoint
			// store newly created endpoints in queriedEndpoints
			queriedEndpoints[newHnsEndpoint.hnsID] = newHnsEndpoint
			queriedEndpoints[newHnsEndpoint.ip] = newHnsEndpoint
		}

		var hnsEndpoints []EndpointsInfo
		/*
			klog.V(4).InfoS("Applying Policy", "serviceInfo", svcName)
			// Create Remote endpoints for every endpoint, corresponding to the service
			containsPublicIP := false
			containsNodeIP := false
		*/

		for _, epInfo := range svcEndpoints.Endpoints {
			fmt.Println(fmt.Sprintf("endpoint -- %v", epInfo))

			var newHnsEndpoint *EndpointsInfo
			hnsNetworkName := "External"
			var err error

			// targetPort is zero if it is specified as a name in port.TargetPort, so the real port should be got from endpoints.
			// Note that hcsshim.AddLoadBalancer() doesn't support endpoints with different ports, so only port from first endpoint is used.
			// TODO(feiskyer): add support of different endpoint ports after hcsshim.AddLoadBalancer() add that.
			//if svcInfo.targetPort == 0 {
			//  svcInfo.targetPort = int(ep.port)
			//	}
			// There is a bug in Windows Server 2019 that can cause two endpoints to be created with the same IP address, so we need to check using endpoint ID first.
			// TODO: Remove lookup by endpoint ID, and use the IP address only, so we don't need to maintain multiple keys for lookup.
			//if len(ep.hnsID) > 0 {
			// newHnsEndpoint = queriedEndpoints[ep.hnsID]
			// }

			ipEp := epInfo.IPs.V4[0]
			if newHnsEndpoint == nil {
				// First check if an endpoint resource exists for this IP, on the current host
				// A Local endpoint could exist here already
				// A remote endpoint was already created and proxy was restarted
				newHnsEndpoint = queriedEndpoints[ipEp]
			}
			if newHnsEndpoint == nil {
				klog.InfoS("Updating network to check for new remote subnet policies", "networkName", hnsNetworkName)
				updatedNetwork, err := hns.getNetworkByName(hnsNetworkName)
				if err != nil {
					klog.ErrorS(err, "Unable to find HNS Network specified, please check network name and CNI deployment", "hnsNetworkName", hnsNetworkName)
					return
				}
				proxier.network = *updatedNetwork
				providerAddress := "192.168.255.20"
				if len(providerAddress) == 0 {
					klog.InfoS("Could not find provider address, assuming it is a public IP", "IP", ipEp)
					providerAddress = proxier.nodeIP.String()
				}

				hnsEndpoint := &EndpointsInfo{
					ip:              ipEp,
					isLocal:         false,
					macAddress:      "00:11:11:22:33:44",
					providerAddress: providerAddress,
				}

				newHnsEndpoint, err = hns.createEndpoint(hnsEndpoint, hnsNetworkName)
				if err != nil {
					klog.ErrorS(err, "Remote endpoint creation failed", "endpointsInfo", hnsEndpoint)
					continue
				}
			} else {
				hnsEndpoint := &EndpointsInfo{
					ip:         ipEp,
					isLocal:    false,
					macAddress: "00:15:5D:C9:6C:8D",
				}

				newHnsEndpoint, err = hns.createEndpoint(hnsEndpoint, hnsNetworkName)
				if err != nil {
					klog.ErrorS(err, "Remote endpoint creation failed")
					continue
				}
			}
			// For Overlay networks 'SourceVIP' on an Load balancer Policy can either be chosen as
			// a) Source VIP configured on kube-proxy (or)
			// b) Node IP of the current node
			//
			// For L2Bridge network the Source VIP is always the NodeIP of the current node and the same
			// would be configured on kube-proxy as SourceVIP
			//
			// The logic for choosing the SourceVIP in Overlay networks is based on the backend endpoints:
			// a) Endpoints are any IP's outside the cluster ==> Choose NodeIP as the SourceVIP
			// b) Endpoints are IP addresses of a remote node => Choose NodeIP as the SourceVIP
			// c) Everything else (Local POD's, Remote POD's, Node IP of current node) ==> Choose the configured SourceVIP
			/*
					providerAddress := proxier.network.findRemoteSubnetProviderAddress(ep.IP())

					isNodeIP := (ep.IP() == providerAddress)
					isPublicIP := (len(providerAddress) == 0)
					klog.InfoS("Endpoint on overlay network", "ip", ep.IP(), "hnsNetworkName", hnsNetworkName, "isNodeIP", isNodeIP, "isPublicIP", isPublicIP)

					containsNodeIP = containsNodeIP || isNodeIP
					containsPublicIP = containsPublicIP || isPublicIP
				}

			*/
			// Save the hnsId for reference
			fmt.Println("Hns endpoint resource", "endpointsInfo", newHnsEndpoint)

			hnsEndpoints = append(hnsEndpoints, *newHnsEndpoint)
			//klog.V(3).InfoS("Endpoint resource found", "endpointsInfo", ep)
		}

		fmt.Println("Associated endpoints for service", "endpointsInfo", hnsEndpoints, "serviceName", svcInfo.Name)
		fmt.Println("Trying to apply Policies for service", "serviceInfo", svcInfo)
		var hnsLoadBalancer *loadBalancerInfo
		//var sourceVip = proxier.sourceVip

		hnsLoadBalancer, err := hns.getLoadBalancer(
			hnsEndpoints,
			loadBalancerFlags{isDSR: proxier.isDSR, isIPv6: proxier.isIPv6Mode},
			"",
			svcip,
			6, //Enum(svcInfo.Ports[0].Protocol),
			uint16(svcInfo.Ports[0].Port),
			uint16(svcInfo.Ports[0].TargetPort),
			queriedLoadBalancers,
		)
		if err != nil {
			klog.ErrorS(err, "Policy creation failed")
			continue
		}

		//svcInfo.hnsID = hnsLoadBalancer.hnsID
		klog.V(3).InfoS("Hns LoadBalancer resource created for cluster ip resources", "clusterIP", svcip, "hnsID", hnsLoadBalancer.hnsID)

		//svcInfo.policyApplied = true
		klog.V(2).InfoS("Policy successfully applied for service", "serviceInfo", svcInfo)
	}
}
