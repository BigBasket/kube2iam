module github.com/jtblin/kube2iam

go 1.16

require (
	github.com/aerospike/aerospike-client-go v4.5.2+incompatible
	github.com/aws/aws-sdk-go-v2 v1.11.0
	github.com/aws/aws-sdk-go-v2/config v1.10.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.10.0
	github.com/coreos/go-iptables v0.6.0
	github.com/gorilla/mux v1.8.0
	github.com/labstack/echo/v4 v4.5.0
	github.com/prometheus/client_golang v1.11.0
	github.com/ryanuber/go-glob v1.0.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9 // indirect
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
)
