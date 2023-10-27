## tanzu config cert add

Add a certificate configuration for a host

### Synopsis

Add a certificate configuration for a host

```
tanzu config cert add [flags]
```

### Examples

```

    # Add CA certificate for a host
    tanzu config cert add --host test.vmware.com --ca-cert path/to/ca/ert

    # Add CA certificate for a host:port
    tanzu config cert add --host test.vmware.com:8443 --ca-cert path/to/ca/ert

    # Set to skip verifying the certificate while interacting with host
    tanzu config cert add --host test.vmware.com  --skip-cert-verify true

    # Set to allow insecure (http) connection while interacting with host
    tanzu config cert add --host test.vmware.com  --insecure true
```

### Options

```
      --ca-cert string            path to the public certificate
  -h, --help                      help for add
      --host string               host or host:port
      --insecure string           allow the use of http when interacting with the host (default "false")
      --skip-cert-verify string   skip server's TLS certificate verification (default "false")
```

### SEE ALSO

* [tanzu config cert](tanzu_config_cert.md)	 - Manage certificate configuration of hosts

