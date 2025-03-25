# Release

### Process

1. Pick a stable (released) version _“v0.X.Y”_ of the operand known to be compatible with operator’s image for _“v0.W.Z_”;
   if needed, [make a release of the operand first](https://github.com/Kuadrant/limitador/blob/main/RELEASE.md).

2. Run the GHA [Release operator](https://github.com/Kuadrant/limitador-operator/actions/workflows/release.yaml); make
   sure to fill all the fields:

    * Branch containing the release workflow file – default: `main`
    * Commit SHA or branch name of the operator to release – usually: `main`
    * Operator version to release (without prefix) – i.e. `0.W.Z`
    * Limitador version the operator enables installations of (without prefix) – i.e. `0.X.Y`
    * If the release is a prerelease

3. Verify that the build [release tag workflow](https://github.com/Kuadrant/limitador-operator/actions/workflows/build-images-for-tag-release.yaml) is triggered and completes for the new tag

4. Verify the new version can be installed from the catalog image, see [Verify OLM Deployment](#verify-olm-deployment)

5. Release to the [community operator index catalogs](#community-operator-index-catalogs).

### Verify OLM Deployment

1. Deploy the OLM catalog image following the [Deploy the operator using OLM](/doc/development.md#deploy-the-operator-using-olm) and providing the generated catalog image. For example:
```sh
make deploy-catalog CATALOG_IMG=quay.io/kuadrant/limitador-operator-catalog:v0.13.0
```

2. Wait for deployment:
```sh
kubectl -n limitador-system wait --timeout=60s --for=condition=Available deployments --all
```

The output should be:

```
deployment.apps/limitador-operator-controller-manager condition met
```

3. Check the logs:
```sh
kubectl -n limitador-system logs -f deployment/limitador-operator-controller-manager
```

4. Check the version of the components deployed:
```sh
kubectl -n limitador-system get deployment -o yaml | grep "image:"
```
The output should be something like:

```
image: quay.io/kuadrant/limitador-operator:v0.13.0
```

### Community Operator Index Catalogs

- [Operatorhub Community Operators](https://github.com/k8s-operatorhub/community-operators)
- [Openshift Community Operators](http://github.com/redhat-openshift-ecosystem/community-operators-prod)

Open a PR on each index catalog ([example](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/1595) |
[docs](https://redhat-openshift-ecosystem.github.io/community-operators-prod/operator-release-process/)).

The usual steps are:

1. Start a new branch named `limitador-operator-v0.W.Z`

2. Create a new directory `operators/limitador-operator/0.W.Z` containing:

    * Copy the bundle files from `github.com/kuadrant/limitador-operator/tree/v0.W.Z/bundle`
    * Copy `github.com/kuadrant/limitador-operator/tree/v0.W.Z/bundle.Dockerfile` with the proper fix to the COPY commands
      (i.e. remove /bundle from the paths)
