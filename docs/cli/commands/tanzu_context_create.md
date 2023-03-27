## tanzu context create

Create a Tanzu CLI context

```
tanzu context create [flags]
```

### Examples

```

	# Create a TKG management cluster context using endpoint
	tanzu context create --endpoint "https://k8s.example.com" --name mgmt-cluster

	# Create a TMC self-managed context using endpoint
	tanzu context create --self-managed --endpoint "https://k8s.example.com" --name test-context

	# Create a TKG management cluster context using kubeconfig path and context
	tanzu context create --kubeconfig path/to/kubeconfig --kubecontext kubecontext --name mgmt-cluster

	# Create a TKG management cluster context using default kubeconfig path and a kubeconfig context
	tanzu context create --kubecontext kubecontext --name mgmt-cluster

	[*] : User has two options to create a kubernetes cluster context. User can choose the control
	plane option by providing 'endpoint', or use the kubeconfig for the cluster by providing
	'kubeconfig' and 'context'. If only '--context' is set and '--kubeconfig' is not set
	$KUBECONFIG env variable would be used and, if $KUBECONFIG env is also not set default
	kubeconfig($HOME/.kube/config) would be used.
	
```

### Options

```
      --endpoint string      endpoint to create a context for
  -h, --help                 help for create
      --kubeconfig string    path to the kubeconfig file; valid only if user doesn't choose 'endpoint' option.(See [*])
      --kubecontext string   the context in the kubeconfig to use; valid only if user doesn't choose 'endpoint' option.(See [*]) 
      --name string          name of the context
  -l, --self-managed         indicate the context is for a self-managed TMC
```

### SEE ALSO

* [tanzu context](tanzu_context.md)	 - Configure and manage contexts for the Tanzu CLI

