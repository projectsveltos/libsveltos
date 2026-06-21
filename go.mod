module github.com/projectsveltos/libsveltos

go 1.26.4

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.22.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.14.0
	github.com/BurntSushi/toml v1.6.0
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3
	github.com/go-logr/logr v1.4.3
	github.com/google/cel-go v0.28.1
	github.com/hexops/gotextdiff v1.0.3
	github.com/mocktools/go-smtp-mock/v2 v2.5.4
	github.com/onsi/ginkgo/v2 v2.31.0
	github.com/onsi/gomega v1.42.0
	github.com/pkg/errors v0.9.1
	github.com/projectsveltos/lua-utils/glua-json v0.0.0-20251212200258-2b3cdcb7c0f5
	github.com/projectsveltos/lua-utils/glua-runes v0.0.0-20251212200258-2b3cdcb7c0f5
	github.com/projectsveltos/lua-utils/glua-sprig v0.0.0-20251212200258-2b3cdcb7c0f5
	github.com/projectsveltos/lua-utils/glua-strings v0.0.0-20251212200258-2b3cdcb7c0f5
	github.com/yuin/gopher-lua v1.1.2
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sync v0.21.0
	golang.org/x/text v0.38.0
	k8s.io/api v0.36.2
	k8s.io/apiextensions-apiserver v0.36.2
	k8s.io/apimachinery v0.36.2
	k8s.io/client-go v0.36.2
	k8s.io/klog/v2 v2.140.0
	k8s.io/utils v0.0.0-20260507154919-ff6756f316d2
	sigs.k8s.io/cluster-api v1.13.2
	sigs.k8s.io/controller-runtime v0.24.1
	sigs.k8s.io/kustomize/api v0.21.1
	sigs.k8s.io/kustomize/kyaml v0.21.1
	sigs.k8s.io/yaml v1.6.0
)

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.2 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/aws/smithy-go v1.27.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260402051712-545e8a4df936 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/cluster-bootstrap v0.36.0 // indirect
	k8s.io/component-base v0.36.2 // indirect
	k8s.io/kube-openapi v0.0.0-20260427204847-8949caaa1199 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.0 // indirect
)
