## tanzu config cert update

Update certificate configuration for a host

```
tanzu config cert update HOST [flags]
```

### Examples

```

    # Update CA certificate for a host,
    tanzu config cert update test.vmware.com --ca-cert path/to/ca/ert

    # Update CA certificate for a host:port,
    tanzu config cert update test.vmware.com:5443 --ca-cert path/to/ca/ert

    # Update whether to skip verifying the certificate while interacting with host
    tanzu config cert update test.vmware.com  --skip-cert-verify true

    # Update whether to allow insecure (http) connection while interacting with host
    tanzu config cert update test.vmware.com  --insecure true
```

### Options

```
      --ca-cert string            path to the public certificate
  -h, --help                      help for update
      --insecure string           allow the use of http when interacting with the host (true|false)
      --skip-cert-verify string   skip server's TLS certificate verification (true|false)
```

### SEE ALSO

* [tanzu config cert](tanzu_config_cert.md)	 - Manage certificate configuration of hosts

