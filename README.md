Terraform Provider Podman
==================

Requirements
------------

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.12.x
-	[Go](https://golang.org/doc/install) >= 1.12

Building The Provider
---------------------

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command: 
```sh
$ go install
```


Using the provider
----------------------

```hcl-terraform
provider "podman" {
  // You may want to authenticate with container registries here
  // If none is given, Docker Hub will be used per default
  registry_auth {
    address = "registry1.com"
    username = "registry1user"
    password = "registry1pass"
  }
  registry_auth {
    address = "registry2.com"
    username = "registry2user"
    password = "registry2pass"
  }
}
```

Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```sh
$ make testacc
```
