<article><H2 dir="auto">cosign-installer GitHub Action</H2> <p dir="auto">This action enables you to sign and verify container images using <code>cosign</code>. <code>cosign-installer</code> verifies the integrity of the <code>cosign</code> release during installation.</p> <p dir="auto">For a quick start guide on the usage of <code>cosign</code>, please refer to <a href="https://github.com/sigstore/cosign#quick-start">https://github.com/sigstore/cosign#quick-start</a>. For available <code>cosign</code> releases, see <a href="https://github.com/sigstore/cosign/releases">https://github.com/sigstore/cosign/releases</a>.</p> <H2 dir="auto">Usage</H2> <p dir="auto">This action currently supports GitHub-provided Linux, macOS and Windows runners (self-hosted runners may not work).</p> <p dir="auto">Add the following entry to your Github workflow YAML file:</p> <pre><code>uses: sigstore/cosign-installer@v4.1.0</code></pre> <p dir="auto">Full example:</p> <pre><code>jobs:
  example:
    runs-on: ubuntu-latest

    permissions: {}

    name: Install Cosign
    steps:
      - name: Install Cosign
        uses: sigstore/cosign-installer@v4.1.0
      - name: Check install!
        run: cosign version</code></pre> <p dir="auto">The used Cosign version only changes when cosign-installer is upgraded. If you need to select a specific Cosign version, use <code>cosign-release</code> but note that you are now responsible for maintaining the Cosign version (in addition to maintaining the cosign-installer version).</p> <p dir="auto">Example pinning Cosign version with <code>cosign-release</code>:</p> <pre><code>jobs:
  example:
    runs-on: ubuntu-latest

    permissions: {}

    name: Install Cosign
    steps:
      - name: Install Cosign
        uses: sigstore/cosign-installer@v4.1.0
        with:
          cosign-release: 'v3.0.6'
      - name: Check install!
        run: cosign version</code></pre> <p dir="auto">If you want to install cosign from its main version by using 'go install' under the hood, you can set 'cosign-release' as 'main'. Once you did that, cosign will be installed via 'go install' which means that please ensure that go is installed.</p> <p dir="auto">Example of installing cosign via go install:</p> <pre><code>jobs:
  example:
    runs-on: ubuntu-latest

    permissions: {}

    name: Install Cosign via go install
    steps:
      - name: Install go
        uses: actions/setup-go@v6.0.0
        with:
          go-version: '1.24'
          check-latest: true
      - name: Install Cosign
        uses: sigstore/cosign-installer@v4.1.0
        with:
          cosign-release: main
      - name: Check install!
        run: cosign version</code></pre> <p dir="auto">This action does not need any GitHub permission to run, however, if your workflow needs to update, create or perform any action against your repository, then you should change the scope of the permission appropriately.</p> <p dir="auto">For example, if you are using the <code>ghcr.io</code> as your registry to push the images you will need to give the <code>write</code> permission to the <code>packages</code> scope.</p> <p dir="auto">Example of a simple workflow:</p> <pre><code>jobs:
  build-image:
    runs-on: ubuntu-latest

    permissions:
      contents: read
      packages: write
      id-token: write # needed for signing the images with GitHub OIDC Token

    name: build-image
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 1

      - name: Install Cosign
        uses: sigstore/cosign-installer@v4.1.0

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3.6.0

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3.11.1

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3.4.0
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - id: docker_meta
        uses: docker/metadata-action@v5.7.0
        with:
          images: ghcr.io/sigstore/sample-honk
          tags: type=sha,format=long

      - name: Build and Push container images
        uses: docker/build-push-action@v6.18.0
        id: build-and-push
        with:
          platforms: linux/amd64,linux/arm/v7,linux/arm64
          push: true
          tags: ${{ steps.docker_meta.outputs.tags }}

      # https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions#using-an-intermediate-environment-variable
      - name: Sign image with a key
        run: |
          images=""
          for tag in ${TAGS}; do
            images+="${tag}@${DIGEST} "
          done
          cosign sign --yes --key env://COSIGN_PRIVATE_KEY ${images}
        env:
          TAGS: ${{ steps.docker_meta.outputs.tags }}
          COSIGN_PRIVATE_KEY: ${{ secrets.COSIGN_PRIVATE_KEY }}
          COSIGN_PASSWORD: ${{ secrets.COSIGN_PASSWORD }}
          DIGEST: ${{ steps.build-and-push.outputs.digest }}

      - name: Sign the images with GitHub OIDC Token
        env:
          DIGEST: ${{ steps.build-and-push.outputs.digest }}
          TAGS: ${{ steps.docker_meta.outputs.tags }}
        run: |
          images=""
          for tag in ${TAGS}; do
            images+="${tag}@${DIGEST} "
          done
          cosign sign --yes ${images}</code></pre> <H3 dir="auto">Optional Inputs</H3> <p dir="auto">The following optional inputs:</p> <div><table> <thead> <tr> <th>Input</th> <th>Description</th> </tr> </thead> <tbody> <tr> <td><code>cosign-release</code></td> <td><code>cosign</code> version to use instead of the default.</td> </tr> <tr> <td><code>install-dir</code></td> <td>directory to place the <code>cosign</code> binary into instead of the default (<code>$HOME/.cosign</code>).</td> </tr> <tr> <td><code>use-sudo</code></td> <td>set to <code>true</code> if <code>install-dir</code> location requires sudo privs. Defaults to false.</td> </tr> </tbody> </table></div>   </article>