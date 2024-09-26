# Custom Image

Currently, the limitador image being used in the deployment is read from different sources with some order of precedence:
* If Limtador CR's `spec.image` is set -> image = `${spec.image}`
* If Limtador CR's `spec.version` is set -> image = `quay.io/kuadrant/limitador:${spec.version}` (note the repo is hardcoded)
* if `RELATED_IMAGE_LIMITADOR` env var is set -> image = `$RELATED_IMAGE_LIMITADOR`
* else: hardcoded to `quay.io/kuadrant/limitador:latest`

The `spec.image` field is not meant to be used in production environments.
It is meant to be used for dev/testing purposes.
The main drawback of the `spec.image` usage is that upgrades cannot be supported as the
limitador operator cannot ensure the operation to be safe.


```yaml
---
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-instance-1
spec:
  image: example.com/myorg/limitador-repo:custom-image-v1
EOF
```

## Pull an Image from a Private Registry

To pull an image from a private container image registry or repository, you need to provide credentials.

Create a Secret by providing credentials on the command line

```
kubectl create secret docker-registry regcred --docker-server=<your-registry-server> --docker-username=<your-name> --docker-password=<your-pword> --docker-email=<your-email>
```

Deploy limitador instance that uses your secret


```yaml
---
apiVersion: limitador.kuadrant.io/v1alpha1
kind: Limitador
metadata:
  name: limitador-instance-1
spec:
  image: example.com/myorg/limitador-repo:custom-image-v1
  imagePullSecrets:
  - name: regcred
```
