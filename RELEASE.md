# How to release Limitador Operator

## Process

To release a version _“v0.W.Z”_ of Limitador Operator in GitHub and Quay.io, follow these steps:

1. Pick a stable (released) version _“v0.X.Y”_ of the operand known to be compatible with operator’s image for _“v0.W.Z_”;
   if needed, [make a release of the operand first](https://github.com/Kuadrant/limitador/blob/main/RELEASE.md).

2. Run the GHA [Release operator](https://github.com/Kuadrant/limitador-operator/actions/workflows/release.yaml); make
   sure to fill all the fields:

    * Branch containing the release workflow file – default: `main`
    * Commit SHA or branch name of the operator to release – usually: `main`
    * Operator version to release (without prefix) – i.e. `0.W.Z`
    * Limitador version the operator enables installations of (without prefix) – i.e. `0.X.Y`
    * Operator replaced version (without prefix) – i.e. `0.P.Q`
    * If the release is a prerelease

3. Run the GHA [Build and push images](https://github.com/Kuadrant/limitador-operator/actions/workflows/build-images-base.yaml)
   for the _“v0.W.Z”_ tag, specifying ‘Limitador version’ equals to _“0.X.Y”_. This will cause the
   new images (bundle and catalog included) to be built and pushed to the corresponding repos in
   [quay.io/kuadrant](https://quay.io/organization/kuadrant).


### Publishing the Operator in OpenShift Community Operators
Open a PR in the [OpenShift Community Operators repo](http://github.com/redhat-openshift-ecosystem/community-operators-prod)
([example](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/1595) |
[docs](https://redhat-openshift-ecosystem.github.io/community-operators-prod/operator-release-process/)).

The usual steps are:

1. Start a new branch named `limitador-operator-v0.W.Z`

2. Create a new directory `operators/limitador-operator/0.W.Z` containing:

    * Copy the bundle files from `github.com/kuadrant/limitador-operator/tree/v0.W.Z/bundle`
    * Copy `github.com/kuadrant/limitador-operator/tree/v0.W.Z/bundle.Dockerfile` with the proper fix to the COPY commands
      (i.e. remove /bundle from the paths)

### Publishing the Operator in Kubernetes Community Operators (OperatorHub.io)

1. Open a PR in the [Kubernetes Community Operators repo](https://github.com/k8s-operatorhub/community-operators)
   ([example](https://github.com/k8s-operatorhub/community-operators/pull/1655) | [docs](https://operatorhub.io/contribute)).

2. The usual steps are the same as for the
   [OpenShift Community Operators](https://docs.google.com/document/d/1tLveyv8Zwe0wKyfUTWOlEnFeMB5aVGqIVDUjVYWax0U/edit#heading=h.b5tapxn4sbk5)
   hub.
