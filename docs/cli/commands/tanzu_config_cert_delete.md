## tanzu config cert delete

Delete certificate configuration for a host

```
tanzu config cert delete [host] [flags]
```

### Examples

```

    # Delete a certificate for host
    tanzu config cert delete test.vmware.com

    # Delete a certificate for host:port
    tanzu config cert delete test.vmware.com:5443
```

### Options

```
  -h, --help   help for delete
```

### SEE ALSO

* [tanzu config cert](tanzu_config_cert.md)	 - Manage certificate configuration of hosts

