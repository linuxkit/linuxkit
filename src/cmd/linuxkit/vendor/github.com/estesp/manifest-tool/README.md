## manifest-tool

`manifest-tool` is a command line utility that implements a portion of the client side of the
Docker registry v2.2 API for interacting with manifest objects in a registry conforming to that
specification.

This tool was mainly created for the purpose of viewing, creating, and pushing the
new **manifests list** object type in the Docker registry.
Manifest lists are defined in the [v2.2 image specification](https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-2.md) and exist mainly
for the purpose of supporting multi-architecture and/or multi-platform images within a Docker registry.

Note that manifest-tool was initially started as a joint project with [Harshal Patil](https://github.com/harche) from IBM Bangalore, and originally forked from the registry client codebase, skopeo, by [Antonio Murdaca/runc0m](https://github.com/runcom), that became a part of [Project Atomic](https://github.com/projectatomic/skopeo) later in its lifetime. Thanks to both Antonio and Harshal for their initial work that made this possible! Also, note that work is ongoing to add these capabilities directly to the Docker client. Thanks to Christy Perez from IBM Systems for her hard work in pushing ahead with [a docker/cli PR](https://github.com/docker/cli/pull/138) which should make this tool obsolete!

> **UPDATE (Feb 2018):** The Docker client PR #138 is merged, providing a `docker manifest`
> command that can replace the use of `manifest-tool` in most scenarios. Other
> follow-on PRs are in process to add functionality to `docker manifest` to make
> it completely functional for all multi-platform image use cases.

### Sample Usage

The two main capabilities of this tool are to 1) **inspect** manifests (of all media types) within any registry supporting the Docker API and 2) to **push** manifest list objects to any registry which supports the Docker V2 API and the v2.2 image specification.

> *Note:* For pushing to an authenticated registry like DockerHub, you will need a config generated via
`docker login`:
```sh
docker login
<enter your credentials>
```
> *Important:* Since 17.03, Docker for Mac stores credentials in the OSX/macOS keychain and not in `config.json`. This means for users of
> `manifest-tool` on Mac, you will need to specify `--username` and `--password` instead of relying on `docker login` credentials. Note that special characters may
> need escaping depending on your shell environment when provided via command line.

If you are pushing to a registry requiring authentication, credentials will be handled as follows:
 - If the `--username` and `--password` flags are supplied, their contents will be used as the basic credentials for any actions requiring authentication.
 - If `--username` and `--password` are **not** provided as command line flags, the default docker client config will be loaded (default location: `$HOME/.docker/config.json`) and credentials for the target registry will be queried if available.
 - If your docker config file is not in the standard location, you can provide an alternate directory location via the `--docker-cfg` flag and the `config.json` file will be used from that alternate directory.

The following example shows the use of `--docker-cfg` to provide an alternate directory location for `docker login`-generated credentials when pushing a manifest list:

```sh
$ ./manifest-tool --docker-cfg '/tmp/some-docker-config/' push from-spec /home/myuser/sample.yml
```

#### Inspect

You can inspect the manifest of any *repo/image:tag* combination by simply using the **inspect** command. Similar to the Docker client, if no tag is provided, *latest* is used as the tag. The output of an inspect on a manifest list media type is shown below. In the case of a manifest list the output shows all the platform-segregated details so the user can easily determine what platforms are supported by the image:

```sh
$ ./manifest-tool inspect trollin/golang:latest
Name:   trollin/golang (Type: application/vnd.docker.distribution.manifest.list.v2+json)
Digest: sha256:e001ca16da8f96ed191624c365de40d964092053a73ffa02667378ecc793dabb
 * Contains 5 manifest references:
1    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
1       Digest: sha256:738ff08d42047a2167c8aed758140d01df835295fa6829f49a45f014045c86b0
1  Mfst Length: 1792
1     Platform:
1           -      OS: linux
1           - OS Vers: 
1           - OS Feat: []
1           -    Arch: amd64
1           - Variant: 
1           - Feature: 
1     # Layers: 7
         layer 1: digest = sha256:9f0706ba7422412cd468804fee456786f88bed94bf9aea6dde2a47f770d19d27
         layer 2: digest = sha256:d3942a742d221ef22a0a335c4eebf09e15a36dcfb224b5a2d0cdcc405f374ccb
         layer 3: digest = sha256:62b1123c88f67a9ad43d9bf3f552bbe3352696a674e82712fda785db4f71a655
         layer 4: digest = sha256:3306e13140efed403fce1789f2760e1356e5b4d76f84b0a92a0ab4b769d3a447
         layer 5: digest = sha256:72bc60ec9d39d021c5af5f1b950d4b5ea15495e36166b662f608bf7272a4ef3c
         layer 6: digest = sha256:2b74c5631cb8592d5eaebfc058c8c0a862d51eff640400452cd1be0089c0b53d
         layer 7: digest = sha256:c3c00cee508294b26a9046e2c43d845e3c4fa9101c449f3f67d696b8e1f0b05a

2    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
2       Digest: sha256:dd7f68050650c90228ee27fb4e6a66a6e55172e6c7bb084c3b257507b1920d9b
2  Mfst Length: 1792
2     Platform:
2           -      OS: linux
2           - OS Vers: 
2           - OS Feat: []
2           -    Arch: arm
2           - Variant: v7
2           - Feature: 
2     # Layers: 7
         layer 1: digest = sha256:72c70f9f7d679945bc71d954dc0c7de236e0067af495d09e9bea24f497cc79b7
         layer 2: digest = sha256:468951d2a7c0c7ac263f85ec984303dc627e7d72cf255d3b496ef2e8820fed0c
         layer 3: digest = sha256:f3ba027ee390db991d1f3721300111af8180e195547e7812b36d992bf1223f8d
         layer 4: digest = sha256:527515a549a64f7c7e59149599a36547266916c3f01f2a520af61731bdc5d84f
         layer 5: digest = sha256:c2a8846df889425dbb7b7eec4aa23ba80b18cb827802fb2c3047f951af8e47eb
         layer 6: digest = sha256:836904189352dbaa66117d2dc671f0248d98af7e5e4fd6c501331e0aadca1fe5
         layer 7: digest = sha256:a420e8eb48d7eaf57cd199b887b205c4a7e772f337abfbb99fb109d800be9cdf

3    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
3       Digest: sha256:7da99e1901b73425d45dc45779784cabd6c68c791cf99a4630177471e95a62ea
3  Mfst Length: 1792
3     Platform:
3           -      OS: linux
3           - OS Vers: 
3           - OS Feat: []
3           -    Arch: 386
3           - Variant: 
3           - Feature: 
3     # Layers: 7
         layer 1: digest = sha256:2fa359c89a0e952ec2fe14e3c584ee13d6ec919c73a7dcac34ba320a459e2a62
         layer 2: digest = sha256:54ddb5e3622b4571bbc0d44b29cebb14d29dbca08a475ff77342f35097464fe9
         layer 3: digest = sha256:c7e97c3de48d315e05dd5fad6a12bfeece80272f6b0c7107bb08d3de60f32f6c
         layer 4: digest = sha256:c19c57d924460ac3012f2e60e9327b31862f5b4410bce307fe18f258dee273c6
         layer 5: digest = sha256:9f2e09461ebc0017a8f53a2a4a770215d72c9c95f40bd84400b39ff321f2fa0d
         layer 6: digest = sha256:af0328d430b279d615d319ccba88420ac3fdff3c9d9ab9ae65ef383613181b12
         layer 7: digest = sha256:48a5a549190d0e2345a3ef52fb36992e4b3b86154648fed6a2455e394339682e

4    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
4       Digest: sha256:71c489123d96fb379a92bb62a696e140eaa24bc44e241e5917dc01f66b22b8cf
4  Mfst Length: 1792
4     Platform:
4           -      OS: linux
4           - OS Vers: 
4           - OS Feat: []
4           -    Arch: ppc64le
4           - Variant: 
4           - Feature: 
4     # Layers: 7
         layer 1: digest = sha256:a5561821dba4ceb47be1d2f5f108a24b391df9d6a3a764d2c04ea8ac29410625
         layer 2: digest = sha256:88807427a2577c993b597a100e9caba7972e266a0a18cba8c2fe2d14f1367764
         layer 3: digest = sha256:82d997bbc6b0b5fec4c8b5fa96e4e89d98bb1ac41bfbecc0682a813fa137e4fb
         layer 4: digest = sha256:b6f51579554854f79e3b930af1eaace3c8a3e9da7df41a8b3bdc97e47697a0ef
         layer 5: digest = sha256:11446b02d18e4448e5359af390493245df61146672600d0be7cfd6e37310ba57
         layer 6: digest = sha256:dd61e5bdd4e1e530175062b0e32515b5c346f43ccf25c081987ae9b6b49c3a15
         layer 7: digest = sha256:7c613dd01f119499d8e7b4b1e4fdb6638f04cd20528e4b3a8e537d61c66cdc18

5    Mfst Type: application/vnd.docker.distribution.manifest.v2+json
5       Digest: sha256:0dc83ed60579807a0e6913e25f403755b733bf2b8415ae633c58b0eff7f53830
5  Mfst Length: 1792
5     Platform:
5           -      OS: linux
5           - OS Vers: 
5           - OS Feat: []
5           -    Arch: s390x
5           - Variant: 
5           - Feature: 
5     # Layers: 7
         layer 1: digest = sha256:29420dd727d39cbedfb85562111f49e24b0b96adda04de4663d2099fbbf4f993
         layer 2: digest = sha256:8df37c45a9ab1f6a86d72a13aaa358015be4fd124c6a11083f75e5371273d5dc
         layer 3: digest = sha256:1533d3ea9025b772c86273232b4ee6a0dd2cb6852dbf3107db1e1aab22b744fd
         layer 4: digest = sha256:7e616db8da0d96fc673928cf73007a7456efcba84a6df127e476a102dfea6f7e
         layer 5: digest = sha256:70c28aac7effb022114051bd9d3df946242c838544b5d2b9c0cc128ff26cffdc
         layer 6: digest = sha256:e6b51fe20471b4195349bec4a2ea07d8b4f2e69cac056f1f41c047d264b23f58
         layer 7: digest = sha256:b3c7c7b9de1680f7f0400d9757a5d8cb5963dbd28d9111ac72da620501bf2f34
```

From this output we can clearly see this is a manifest list object (the media type is output as well) that has five platform definitions to support amd64, i386, s390x (z Systems), and ARMv7. To read more about manifest lists and how the Docker engine uses this information to determine what image/layers to pull read this [blog post on multi-platform support in Docker](https://integratedcode.us/2016/04/22/a-step-towards-multi-platform-docker-images/).

#### Create/Push

Given that the Docker client does not have a way to perform the creation/pushing of manifest list objects (although see the note above regarding the in-process PR to correct this), the main role of `manifest-tool` is to create manifest list entries and push them to a Docker registry v2.2 API-supporting repository. The classic method to define the manifest list particulars is via a YAML file.

A sample YAML file is shown below. The cross-repository push feature is exploited in `manifest-tool`
so that the source and target image names can differ as long as they are within the same registry.
For example, a source image could be named `myprivreg:5000/someimage_ppc64le:latest` and 
referenced by a manifest list in repository  `myprivreg:5000/someimage:latest`.

With a private registry running on port 5000, a sample YAML input to create a manifest list
combining a ppc64le and amd64 image would look like this:
```
image: myprivreg:5000/someimage:latest
manifests:
  -
    image: myprivreg:5000/someimage:ppc64le
    platform:
      architecture: ppc64le
      os: linux
  -
    image: myprivreg:5000/someimage:amd64
    platform:
      architecture: amd64
      features:
        - sse
      os: linux
```

With the above YAML definition, creating the manifest list with the tool would use the following command:

```sh
$ ./manifest-tool push from-spec someimage.yaml
```

In addition to the YAML file format, `manifest-tool` has the option to use command line arguments to provide the specified images/tags and platform OS/architecture details. Instead of `from-spec` you can use `from-args` with the following format:

```
$ ./manifest-tool push from-args \
    --platforms linux/amd64,linux/arm,linux/arm64 \
    --template foo/bar-ARCH:v1 \
    --target foo/bar:v1
```

On the command line you specify the platform os/arch pairs, a template for finding the source images for each input platform pair, and a target image name.

Specifically:
 - `--platforms` specifies which platforms you want to push for in the form OS/ARCH,OS/ARCH,...
 - `--template` specifies the image repo:tag source for inputs by replacing the placeholders `OS` and `ARCH` with the inputs from `--platforms`.
 - `--target` specifies the target image repo:tag that will be the manifest list entry in the registry.

##### Functional Changelog for Push/Create

 - Release **v0.5.0**:
  1. You can now specify `--ignore-missing` and if any of the input images are not available, the tool will output a warning but will not terminate. This allows for "best case" creation of manifest lists based on available images at the time.
  2. Using the YAML input option, you can leave the platform specification empty and `manifest-tool` will auto-populate the platform definition by using the source image manifest OS/arch details. Note that this is potentially deficient for cases where the image was built in a cross-compiled fashion and the source image data is incorrect as it does not match the binary OS/arch content in the image layers.

 - Release **v0.6.0**:
  1. You can specify `tags:` as a list of additional tags to push to the registry against the target manifest list name being created ([#32](https://github.com/estesp/manifest-tool/pull/32)):

```yaml
image: myprivreg:5000/someimage:1.0.0
tags: ['1.0', '1', 'latest']
manifests:
  ...
```

 - Release **v0.7.0**:
  1. The output of `manifest-tool` was modified to add the size of the manifest list canonical JSON pushed to the registry. This allows manifest list content to be signed using 3rd party tools like `notary` which needs the size of the object to validate and sign the content. This is used by the [LinuxKit project](https://github.com/linuxkit/linuxkit) to create signed manifest lists of all of their container images. Example output at the end of a successful manifest list create is shown below. Note that the size field is appended to the digest hash in this version:
```
Digest: sha256:f316f43aceb7a920a7b6c0278c76694a84f608b72bd955db7c9e24927e7edcb3 2058
```

### Building

The releases of `manifest-tool` are built using the latest Go version; currently 1.12.x.

To build `manifest-tool`, clone this repository into your `$GOPATH`:

```sh
$ cd $GOPATH/src
$ mkdir -p github.com/estesp
$ cd github.com/estesp
$ git clone https://github.com/estesp/manifest-tool
$ cd manifest-tool && make binary
```

If you do not have a local Golang environment, you can use the `make build` target to build `manifest-tool` in a Golang 1.9.1-based container environment. This will require that you have Docker installed. The `make static` target will build a statically-linked binary, and `make cross` is used to build all supported CPU architectures, creating static binaries for each platform.

Note that signed binary releases are available on the project's [GitHub releases page](https://github.com/estesp/manifest-tool/releases) for several CPU architectures for Linux as well as OSX/macOS.

### Using manifest-tool Without Installation

Interested in using `manifest-tool` for simple query operations? For example,
maybe you only want to query if a specific image:tag combination is a manifest
list entry or not, and if so, what platforms are listed in the manifest.

You can consume this feature of `manifest-tool` without installing the binary
as long as you are querying public (e.g. not private/authentication-requiring
registries) images via another project, [mquery](https://github.com/estesp/mquery).

You can use `mquery` via a multi-platform image currently located on DockerHub
as **mplatform/mquery:latest**. For example, you can query the `mquery` image
itself with the following command

```sh
$ docker run --rm mplatform/mquery mplatform/mquery
Image: mplatform/mquery
 * Manifest List: Yes
 * Supported platforms:
   - linux/amd64
   - linux/arm/undefined
   - linux/arm64/undefined
   - linux/ppc64le
   - linux/s390x
   - windows/amd64:10.0.14393.1593
```

Note that the `undefined` reference in the output is due to the fact that
the variant field isn't being filled out in the manifest list platform
object for this image.

The `mquery` program itself is a small Go program that queries functions
running via [OpenWhisk](http://openwhisk.incubator.apache.org/) in [IBM Cloud Functions](https://console.bluemix.net/docs/openwhisk/index.html#getting-started-with-cloud-functions) public serverless offering. One
of those functions is packaged as a Docker container image with
`manifest-tool` installed. More information is available in the
[mquery GitHub repo](https://github.com/estesp/mquery). You can read more
of the background details in [my blog post about the Moby Summit EU talk](https://integratedcode.us/2017/11/21/moby-summit-serverless-openwhisk-multi-arch/)
on this topic.

### Known Supporting Registries

Not every registry that claims Docker v2 image API and format support allows manifest lists to be pushed. The errors are not always clear; it could be blocking the blob mount API calls, or the push of the manifest list media type object. At this point, the good news is that the growth of interest in manifest list images has caused quite a few popular registries to add or fix Docker v2 API and image support for manifest lists. The following is a known list of publicly available Docker v2 conformant registries which have been tested with `manifest-tool` or `docker manifest`:

 1. [DockerHub](https://hub.docker.com): Has supported manifest lists since 2016.
 2. [Google Container Registry/gcr.io](https://cloud.google.com/container-registry/): gcr.io manifest list support was fixed in 4Q2017.
 3. [IBM Cloud Container Registry](https://www.ibm.com/cloud/container-registry): The IBM public cloud container registry supports manifest lists since the latter half of 2017.
 4. [Microsoft Azure Container Registry/azurecr.io](https://azure.microsoft.com/en-us/services/container-registry/): The Azure CR supports manifest lists.

### Test a Registry for "Manifest List" Support

If you operate or use a registry claiming conformance to the Docker distribution v2 API and v2.2 image
specification you may want to confirm that this image registry supports the manifest list *media type*
and the APIs used to create a manifest list.

This GitHub repo now has a pre-configured test script which will use readily available multi-architecture
content from DockerHub and tag, push, and then combine it into a manifest list against any image registry
you point it to. See the [test-registry.sh script](https://github.com/estesp/manifest-tool/blob/master/integration/test-registry.sh) in this repo's **integration** directory
for further details. A simple use of the script is
shown below to test a private registry:
```
$ ./test-registry.sh r.myprivreg.com/somerepo
```

> **Note:** This script will expect login details
> have already been provided to `docker login` and
> will use those stored credentials for push and
> API access to *somerepo* on *r.myprivreg.com*.

### License

`manifest-tool` is licensed under the Apache Software License (ASL) 2.0
