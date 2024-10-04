## tanzu context create

Create a Tanzu CLI context

```
tanzu context create CONTEXT_NAME [flags]
```

### Examples

```

    # Create a TKG management cluster context using endpoint and type (--type is optional, if not provided the CLI will infer the type from the endpoint)
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --type k8s

    # Create a TKG management cluster context using endpoint
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port]

    # Create a TKG management cluster context using kubeconfig path and context
    tanzu context create mgmt-cluster --kubeconfig path/to/kubeconfig --kubecontext kubecontext

    # Create a TKG management cluster context by using the provided CA Bundle for TLS verification:
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create a TKG management cluster context by explicit request to skip TLS verification, which is insecure:
    tanzu context create mgmt-cluster --endpoint https://k8s.example.com[:port] --insecure-skip-tls-verify

    # Create a TKG management cluster context using default kubeconfig path and a kubeconfig context
    tanzu context create mgmt-cluster --kubecontext kubecontext

    # Create a TMC(mission-control) context using endpoint and type 
    tanzu context create mytmc --endpoint tmc.example.com:443 --type tmc

    # Create a Tanzu context with the default endpoint (--type is not necessary for the default endpoint)
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com

    # Create a Tanzu context (--type is needed for a non-default endpoint)
    tanzu context create mytanzu --endpoint https://non-default.tanzu.endpoint.com --type tanzu

    # Create a Tanzu context by using the provided CA Bundle for TLS verification:
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com  --endpoint-ca-certificate /path/to/ca/ca-cert

    # Create a Tanzu context but skip TLS verification (this is insecure):
    tanzu context create mytanzu --endpoint https://api.tanzu.cloud.vmware.com --insecure-skip-tls-verify

    Notes: 
    1. TMC context: To create Mission Control (TMC) context an API Key is required. It can be provided using the
       TANZU_API_TOKEN environment variable or entered during context creation.
    2. Tanzu context: To create Tanzu context an API Key is optional. If provided using the TANZU_API_TOKEN environment
       variable, it will be used. Otherwise, the CLI will attempt to log in interactively to the user's default Cloud Services
       organization. You can override or choose a custom organization by setting the TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID
       environment variable with the custom organization ID value. More information regarding organizations in Cloud Services
       and how to obtain the organization ID can be found at
       https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html
       Also, more information on creating tanzu context and using interactive login in terminal based hosts (without browser) can be found at
       https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/quickstart/quickstart.md#creating-and-connecting-to-a-new-context

    [*] : Users have two options to create a kubernetes cluster context. They can choose the control
    plane option by providing 'endpoint', or use the kubeconfig for the cluster by providing
    'kubeconfig' and 'context'. If only '--context' is set and '--kubeconfig' is not, the
    $KUBECONFIG env variable will be used and, if the $KUBECONFIG env is also not set, the default
    kubeconfig file ($HOME/.kube/config) will be used.
```

### Options

```
      --endpoint string                  endpoint to create a context for
      --endpoint-ca-certificate string   path to the endpoint public certificate
  -h, --help                             help for create
      --insecure-skip-tls-verify         skip endpoint's TLS certificate verification
      --kubeconfig string                path to the kubeconfig file; valid only if user doesn't choose 'endpoint' option.(See [*])
      --kubecontext string               the context in the kubeconfig to use; valid only if user doesn't choose 'endpoint' option.(See [*]) 
  -t, --type string                      type of context to create (kubernetes[k8s]/mission-control[tmc]/tanzu)
```

### SEE ALSO

* [tanzu context](tanzu_context.md)	 - Configure and manage contexts for the Tanzu CLI

