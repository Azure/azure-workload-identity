module github.com/Azure/azure-workload-identity

go 1.19

require (
	github.com/Azure/aad-pod-identity v1.8.13
	github.com/Azure/azure-sdk-for-go v65.0.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v0.13.2
	github.com/Azure/go-autorest/autorest v0.11.28
	github.com/Azure/go-autorest/autorest/adal v0.9.21
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/AzureAD/microsoft-authentication-library-for-go v0.7.0
	github.com/go-logr/logr v1.2.3
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mattn/go-colorable v0.1.13
	github.com/microsoft/kiota/abstractions/go v0.0.0-20211202082735-099f3c37853a
	github.com/microsoft/kiota/authentication/go/azure v0.0.0-20211201125630-3501743a5dc5
	github.com/microsoft/kiota/serialization/go/json v0.0.0-20211112084539-17ac73ffdc7c
	github.com/microsoftgraph/msgraph-beta-sdk-go v0.7.0
	github.com/onsi/ginkgo/v2 v2.1.6
	github.com/onsi/gomega v1.20.2
	github.com/open-policy-agent/cert-controller v0.5.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.2
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	gopkg.in/ini.v1 v1.62.1
	gopkg.in/square/go-jose.v2 v2.6.0
	k8s.io/api v0.25.3
	k8s.io/apimachinery v0.25.3
	k8s.io/client-go v0.25.3
	k8s.io/klog/v2 v2.80.1
	k8s.io/kubernetes v1.25.3
	k8s.io/utils v0.0.0-20220823124924-e9cbc92d1a73
	sigs.k8s.io/controller-runtime v0.13.0
)

require (
	cloud.google.com/go v0.98.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.21.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v0.9.1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/go-logr/zapr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/microsoft/kiota/http/go/nethttp v0.0.0-20211203130928-8449c9e67101 // indirect
	github.com/microsoftgraph/msgraph-sdk-go-core v0.0.5 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pkg/browser v0.0.0-20210115035449-ce105d075bb4 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.1 // indirect
	go.opentelemetry.io/contrib v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v0.20.0 // indirect
	go.opentelemetry.io/proto/otlp v0.7.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa // indirect
	golang.org/x/net v0.0.0-20220906165146-f3363e06e74c // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220609170525-579cf78fd858 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21 // indirect
	google.golang.org/grpc v1.47.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.25.3 // indirect
	k8s.io/apiserver v0.25.3 // indirect
	k8s.io/component-base v0.25.3 // indirect
	k8s.io/component-helpers v0.25.3 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	k8s.io/kubectl v0.0.0 // indirect
	k8s.io/kubelet v0.0.0 // indirect
	k8s.io/pod-security-admission v0.0.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.33 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.25.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.25.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.3
	k8s.io/apiserver => k8s.io/apiserver v0.25.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.25.3
	k8s.io/client-go => k8s.io/client-go v0.25.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.25.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.25.3
	k8s.io/code-generator => k8s.io/code-generator v0.25.3
	k8s.io/component-base => k8s.io/component-base v0.25.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.25.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.25.3
	k8s.io/cri-api => k8s.io/cri-api v0.25.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.25.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.25.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.25.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.25.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.25.3
	k8s.io/kubectl => k8s.io/kubectl v0.25.3
	k8s.io/kubelet => k8s.io/kubelet v0.25.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.25.3
	k8s.io/metrics => k8s.io/metrics v0.25.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.25.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.25.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.25.3
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.25.3
	k8s.io/sample-controller => k8s.io/sample-controller v0.25.3
)
