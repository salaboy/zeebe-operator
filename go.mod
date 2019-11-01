module zeebe-operator

go 1.12

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/go-containerregistry v0.0.0-20191029173801-50b26ee28691 // indirect
	github.com/google/uuid v1.1.1
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/tektoncd/pipeline v0.8.0
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898 // indirect
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	knative.dev/pkg v0.0.0-20191031034812-340e3aefcd4b // indirect
	sigs.k8s.io/controller-runtime v0.2.2
	sigs.k8s.io/controller-tools v0.2.1 // indirect
)
