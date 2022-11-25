## tanzu cluster machinehealthcheck delete

Delete a MachineHealthCheck object of a cluster

### Synopsis

Delete a MachineHealthCheck object of a cluster

```
tanzu cluster machinehealthcheck delete CLUSTER_NAME [flags]
```

### Options

```
  -h, --help               help for delete
  -m, --mhc-name string    Name of the MachineHealthCheck object
  -n, --namespace string   The namespace where the MachineHealthCheck object was created, default to the cluster's namespace
  -y, --yes                Delete the MachineHealthCheck object without asking for confirmation
```

### Options inherited from parent commands

```
      --log-file string   Log file path
  -v, --verbose int32     Number for the log level verbosity(0-9)
```

### SEE ALSO

* [tanzu cluster machinehealthcheck](tanzu_cluster_machinehealthcheck.md)     - MachineHealthCheck operations for a cluster

###### Auto generated by spf13/cobra on 15-Jul-2021