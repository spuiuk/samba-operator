
# Create a simple share

The simplest thing one can do with the operator is to create a single SmbShare
with an embedded PVC spec:

```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: myshare
  namespace mynamespace
spec:
  readOnly: false
  storage:
    pvc:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```

This will create a share with a single user named `sambauser` and the password
`samba`.  This user is good for demos, but not much else. :-)


# Giving a share a custom name

SMB shares support a wider variety of names than Kubernetes does for resources.
By default the operator names the SMB share after the SmbShare resource.
However, you can specify a name to be used.

```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: myshare
  namespace mynamespace
spec:
  shareName: "My Great Share"
  readOnly: false
  storage:
    pvc:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```

Now the name of the SMB share will be "My Great Share" instead of "myshare".
Note that this only changes the name of the share in the SMB protocol.
Resources created by the operator and other networking aspects may continue to
reflect the name of the SmbShare resource.

# Export a path within a pre-existing PVC.

Given a pre-existing PVC `mypvc` containing a directory `exports` which is to be exported as a share.

```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: smbshare1
spec:
  storage:
    pvc:
      name: "mypvc"
      path: "exports"
  readOnly: false
```

The `path` directive only supports the export of top level directories within the PVC.

# Configure a share with custom users

This example updates the share from the previous example by adding a reference
to an SmbSecurityConfig resource. This resource tells the servers backing each
share that it should create users for access. The SmbSecurityConfig resource
refers to a Kubernetes secret that will store JSON defining each user. You can
create multiple SmbShare resources that all use the same users or you can
create unique SmbSecurityConfig instances for each share.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: users1
  namespace mynamespace
type: Opaque
stringData:
  demousers: |
    {
      "samba-container-config": "v0",
      "users": {
        "all_entries": [
          {
            "name": "user1",
            "password": "T0Psecre7"
          },
          {
            "name": "user2",
            "password": "L37me1N"
          }
        ]
      }
    }
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbSecurityConfig
metadata:
  name: myusers
  namespace mynamespace
spec:
  mode: user
  users:
    secret: users1
    key: demousers
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: myshare
  namespace mynamespace
spec:
  securityConfig: myusers
  readOnly: false
  storage:
    pvc:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```


# Configure a share for Active Directory based authentication

This example is similar to the user authentication example, but instead of
configuring the SmbSecurityConfig resource to indicate that (standalone) user
and group security should be used we configure it for Active Directory based
security. Much like the previous example this example also requires a secret.
In this case, the secret specifies a user and password (see NOTE) that can join
resources to the AD Domain. In the SmbSecurityConfig the `realm:` field
specifies the name of the domain/realm to join and the joinSources specifies
what secret and key holds the information needed to join Active Directory.
Finally, the SmbShare references our SmbSecurityConfig by name.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: join1
  namespace mynamespace
type: Opaque
stringData:
  # Change the value below to match the username and password for a user that
  # can join systems your test AD Domain
  join.json: |
    {"username": "Administrator", "password": "P4ssw0rd"}
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbSecurityConfig
metadata:
  name: mydomain
  namespace mynamespace
spec:
  mode: active-directory
  realm: cooldomain.myorg.example.com
  joinSources:
  - userJoin:
      secret: join1
      key: join.json
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: myshare
  namespace mynamespace
spec:
  securityConfig: mydomain
  readOnly: false
  storage:
    pvc:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```

NOTE: Currently, we only support joining to Active Directory by username &
password.  We're aware that storing AD credentials in a Kubernetes resource
may not be everyone's first choice. We're looking into other methods in the
future. Do note that by separating the credentials in the secret, the password
is never directly accessed by the operator itself.


# Use a cluster internal share as persistent volume (without auto-provisioning)

Kubernetes does not by default support mounting SMB shares.  For now
it's neccessary to install the [smb csi
driver](https://github.com/kubernetes-csi/csi-driver-smb).  Please refer
to its documentation on how to install the csi driver.

First create a (non provisioning) storage class to differentiate SMB shares
from other storage.

```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: smb
parameters:
  type: smb
provisioner: kubernetes.io/no-provisioner
reclaimPolicy: Retain
volumeBindingMode: Immediate
```

When using Active Directory, the username and password to mount the share must match a
username/password pair that exists in your AD. When using pre-defined users &
groups the username/password pair must match one that is defined in the JSON
embedded in the secret associated with your SmbSecurityConfig.

```
apiVersion: v1
kind: Secret
metadata:
  name: myshare-mount-creds
  namespace mynamespace
type: Opaque
stringData:
  username: user1
  password: T0Psecre7
```

The following persistent volume will allow mounting the share.
Note the `spec.csi.volumeAttributes.source`: `myshare` is the share's service name, `mynamespace` the namespace the `SmbShare` is in and `My Great Share` is the share's `shareName` as configured or the share's name if not.

```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-mynamespace-myshare
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  mountOptions:
    - dir_mode=0777
    - file_mode=0777
    - vers=3.0
  csi:
    driver: smb.csi.k8s.io
    readOnly: false
    volumeHandle: mynamespace-myshare  # make sure it's a unique id in the cluster
    volumeAttributes:
      source: "//myshare.mynamespace/My Great Share"
    nodeStageSecretRef:
      name: myshare-mount-creds
      namespace: mynamespace
  claimRef:
    apiVersion: v1
    kind: PersistentVolumeClaim
    name: myshare-smb
    namespace: mynamespace
```

Then the volume claim can be created and should bind shorty after to the persistent volume.
```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: myshare-smb
  namespace mynamespace
spec:
  accessModes:
  - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
  storageClassName: smb
  volumeMode: Filesystem
  volumeName: mynamespace-myshare-smb
```


# Create shares that are accessible outside the cluster

Unless you took extra steps on your own, the shares created in the previous
examples are only accessible by other processes running within the same
Kubernetes cluster.  The Samba Operator does support a simple method for
automatically exposing the shares outside of the cluster, this involves
creating an SmbCommonConfig resource.  Create an SmbCommonConfig like the one
below - that specifies the value "external" for the `publish:` key under
`network:`. Then create an SmbShare that refers to this in it's `commonConfig:`
field. When the operator processes this SmbShare, it will also create a
Kubernetes Service configured for load balancing.

```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbCommonConfig
metadata:
  name: mypublished
  namespace mynamespace
spec:
  network:
    publish: external
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: myshare
  namespace mynamespace
spec:
  commonConfig: mypublished
  readOnly: false
  storage:
    pvc:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```

When the [LoadBalancer
Service](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer)
is created it will report the IP/hostname that you can use to access the share
when you run `kubectl get services`.


# Create shares accessible outside the cluster with DNS registration

This example is like the previous but includes automatically registering the
external IP address used to access the share in the Active Directory DNS. This
means that hosts connected to the AD DNS do not need to be told the IP of the
LoadBalancer Service. This is only supported on shares enabled for Active
Directory.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: join1
  namespace mynamespace
type: Opaque
stringData:
  # Change the value below to match the username and password for a user that
  # can join systems your test AD Domain
  join.json: |
    {"username": "Administrator", "password": "P4ssw0rd"}
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbSecurityConfig
metadata:
  name: mydomain
  namespace mynamespace
spec:
  mode: active-directory
  realm: cooldomain.myorg.example.com
  joinSources:
  - userJoin:
      secret: join1
      key: join.json
  dns:
    register: external-ip
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbCommonConfig
metadata:
  name: mypublished
  namespace mynamespace
spec:
  network:
    publish: external
```
```yaml
apiVersion: samba-operator.samba.org/v1alpha1
kind: SmbShare
metadata:
  name: myshare
  namespace mynamespace
spec:
  securityConfig: mydomain
  commonConfig: mypublished
  readOnly: false
  storage:
    pvc:
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
```

Once a pod exists to serve the share you should be able to resolve a name like
`<share-resource-name>.<yourdomain>`. Using the examples above this would be:
`myshare.cooldomain.myorg.example.com`.
