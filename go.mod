module github.com/lyraproj/hiera

require (
	github.com/Azure/azure-sdk-for-go v32.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.4.0
	github.com/Azure/go-autorest/autorest/adal v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.1.0 // indirect
	github.com/bmatcuk/doublestar v1.1.1
	github.com/hashicorp/go-hclog v0.9.0
	github.com/hashicorp/terraform v0.12.6
	github.com/labstack/echo v3.3.10+incompatible
	github.com/labstack/gommon v0.2.9 // indirect
	github.com/lyraproj/issue v0.0.0-20190606092846-e082d6813d15
	github.com/lyraproj/pcore v0.0.0-20191009094231-d24a2ffc5639
	github.com/spf13/cobra v0.0.4
	github.com/stretchr/testify v1.3.0
	github.com/zclconf/go-cty v1.0.1-0.20190708163926-19588f92a98f
	gopkg.in/yaml.v3 v3.0.0-20190905181640-827449938966
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.3.0+incompatible
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.13+incompatible
	github.com/ugorji/go => github.com/ugorji/go v0.0.0-20181204163529-d75b2dcb6bc8
)

go 1.12
