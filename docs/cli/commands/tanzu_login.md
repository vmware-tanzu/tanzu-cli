## tanzu login

Login to Tanzu Platform for Kubernetes

```
tanzu login [flags]
```

### Examples

```

    # Login to Tanzu
    tanzu login

    # Login to Tanzu using non-default endpoint
    tanzu login --endpoint "https://login.example.com"

    # Login to Tanzu by using the provided CA Bundle for TLS verification
    tanzu login --endpoint https://test.example.com[:port] --endpoint-ca-certificate /path/to/ca/ca-cert

    # Login to Tanzu by explicit request to skip TLS verification (this is insecure)
    tanzu login --endpoint https://test.example.com[:port] --insecure-skip-tls-verify

    Note:
       To login to Tanzu an API Key is optional. If provided using the TANZU_API_TOKEN environment
       variable, it will be used. Otherwise, the CLI will attempt to log in interactively to the user's default Cloud Services
       organization. You can override or choose a custom organization by setting the TANZU_CLI_CLOUD_SERVICES_ORGANIZATION_ID
       environment variable with the custom organization ID value. More information regarding organizations in Cloud Services
       and how to obtain the organization ID can be found at
       https://docs.vmware.com/en/VMware-Cloud-services/services/Using-VMware-Cloud-Services/GUID-CF9E9318-B811-48CF-8499-9419997DC1F8.html
       Also, more information on logging into Tanzu Platform Platform for Kubernetes and using
       interactive login in terminal based hosts (without browser) can be found at
       https://github.com/vmware-tanzu/tanzu-cli/blob/main/docs/quickstart/quickstart.md#logging-into-tanzu-platform-for-kubernetes

```

### Options

```
      --endpoint string                  endpoint to login to (default "https://api.tanzu.cloud.vmware.com")
      --endpoint-ca-certificate string   path to the endpoint public certificate
  -h, --help                             help for login
      --insecure-skip-tls-verify         skip endpoint's TLS certificate verification
```

### SEE ALSO

* [tanzu](tanzu.md)	 - The Tanzu CLI

