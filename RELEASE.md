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

3. Verify that the build [release tag workflow](https://github.com/Kuadrant/dns-operator/actions/workflows/build-images-for-tag-release.yaml) is triggered and completes for the new tag

4. Verify the new version can be installed from the catalog image, see [Verify OLM Deployment](#verify-olm-deployment)

5. Release to the [community operator index catalogs](#community-operator-index-catalogs).

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
