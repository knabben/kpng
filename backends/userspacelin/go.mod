module sigs.k8s.io/kpng/backends/userspacelin

go 1.18

replace (
	k8s.io/apiserver => k8s.io/apiserver v0.22.4 // indirect
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.4
	k8s.io/client-go => k8s.io/client-go v0.22.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.4
	k8s.io/code-generator => k8s.io/code-generator v0.22.4
	k8s.io/component-base => k8s.io/component-base v0.22.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.4
	k8s.io/cri-api => k8s.io/cri-api v0.22.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.4
	k8s.io/kubectl => k8s.io/kubectl v0.22.4
	k8s.io/kubelet => k8s.io/kubelet v0.22.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.4
	k8s.io/metrics => k8s.io/metrics v0.0.0-20210821163913-98d2fd1dc73d
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.4
)

require (
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/klog/v2 v2.20.0
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
)

require sigs.k8s.io/kpng/api v0.0.0-20220521134046-f747cedbe766

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220107163113-42d7afdf6368 // indirect
	google.golang.org/grpc v1.41.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
