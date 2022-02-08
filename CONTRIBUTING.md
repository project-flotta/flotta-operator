# Contributing guidelines

## Formatting

By default, formating is checked by gofmt tool, where almost format all the
cases.

The project does not have a line length limit, but the team try to be as small
as possible, keeping the code clean on reading. Having an 80-120 char limit is
an excellent number to split the line.

## Style

The project follows the Golang project guidelines that you can find in the
following link:

[https://github.com/golang/go/wiki/CodeReviewComments]()

## Developer environment

By default, all Flotta projects rely on Makefiles, where users can run all the
workflows operations.

To make it easier, all Makefiles implement a help section to see the actions you
can run. Here is an example:

```
--> make help
Usage:
  make <target>

General
  help             Display this help.

Development
  manifests        Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
  generate         Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
  fmt              Run go fmt against code.
  vet              Run go vet against code.
  test             Run tests.

Build
  build            Build manager binary.
  run              Run a controller from your host.
  docker-build     Build docker image with the manager.
  docker-push      Push docker image with the manager.

Deployment
  install          Install CRDs into the K8s cluster specified in ~/.kube/config.
  uninstall        Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
  deploy           Deploy controller to the K8s cluster specified in ~/.kube/config.
  undeploy         Undeploy controller from the K8s cluster specified in ~/.kube/config.
  controller-gen   Download controller-gen locally if necessary.
  kustomize        Download kustomize locally if necessary.
```

The core maintainers are developing this project using the latest Fedora and
RHEL distributions.  If you're using any other distribution, maybe you can hit
any not know issue.

### Testing:

All projects implement the target `make test`, where dependencies should be
checked before running the test.

For testing, Ginkgo and Gomega is the prefered flavour. You can find multiple
examples in the current project. The test rules are the following:
- Test Suite should be run on different packages.
- Gomock is the prefered flavour for mocking.
- Should avoid any Conditionals or logic.
- Common code should be moved as helper functions.
- [Table Driven](https://onsi.github.io/ginkgo/#table-driven-tests) tests are
  recommended
- Test structure should follow the following code-comments
```
  It("Can retrieve message correctly", func() {
    // given
    device := getDevice("foo")
    edgeDeviceRepoMock.EXPECT().
      Read(gomock.Any(), "foo", testNamespace).
      Return(device, nil).
      Times(1)

    // when
    res := handler.GetControlMessageForDevice(context.TODO(), params)

    // then
    Expect(res).To(Equal(operations.NewGetControlMessageForDeviceOK()))
  })
```
Projects are continuously tested, and the main branch should constantly be
working.

## Pull Request process

1) Fork the project and commit to a local branch.
2) Submit a PR with all details. Small PRs are prefered; on larger ones, please
ask any maintainers before a significant change to be aligned with the project
roadmap. DCO is needed.
3) The PR to be approved should contain test cases on the new features added.
4) Maintainer will approve the GH actions checks.
5) If all checks are working, PR will be merged.  (Checks can be found on
`.github` folder)

### Contributor compliance with Developer Certificate Of Origin (DCO)

We require every contributor to certify that they are legally permitted to
contribute to our project.  A contributor expresses this by consciously signing
their commits, and by this act expressing that they comply with the [Developer
Certificate Of Origin](https://developercertificate.org/).

A signed commit is a commit where the commit message contains the following
content:

```
Signed-off-by: John Doe <jdoe@example.org>
```

This can be done by adding
[`--signoff`](https://git-scm.com/docs/git-commit#Documentation/git-commit.txt---signoff)
to your git command line.

## Documentation

Currently, all the docs can be found on the flotta-operator project; under the
[docs](docs) folder, in there you can find:

- Architecture details
- API documentation.
- How to deploy workloads.
- Troubleshooting documentation.

## Security

Due to the early stage of the project, security details will be implemented when
the project reaches beta releases.
