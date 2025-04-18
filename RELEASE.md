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

4. Verify the new version can be installed from the catalog image.

   4.1. Deploy the new OLM catalog image
   Create kind cluster
   ```sh
   make kind-create-cluster
   ```
   Deploy OLM system
   ```sh
   make install-olm
   ```
   Deploy the catalog image. Replace `<NEW_TAG>` with the new release tag.
   ```sh
   make deploy-catalog CATALOG_IMG=quay.io/kuadrant/limitador-operator-catalog:<NEW_TAG> DEFAULT_CHANNEL=stable
   ```
   4.2. Wait for deployment:
   ```sh
   kubectl -n limitador-system wait --timeout=60s --for=condition=Available deployments --all
   ```
    The output should be:
   ```
   deployment.apps/limitador-operator-controller-manager condition met
   ```
   4.3. Check the logs:
   ```sh
   kubectl -n limitador-system logs deployment/limitador-operator-controller-manager
   ```
   4.4. Check the version of the components deployed:
   ```sh
   kubectl -n limitador-system get deployment -o yaml | grep "image:"
   ```
   The output should be something like:
   ```
   image: quay.io/kuadrant/limitador-operator:v0.13.0
   ```
