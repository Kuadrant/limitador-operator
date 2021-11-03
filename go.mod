module github.com/kuadrant/limitador-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.15.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/klog/v2 v2.2.0
	sigs.k8s.io/controller-runtime v0.7.2
)
