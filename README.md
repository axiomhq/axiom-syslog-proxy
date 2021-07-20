# Axiom Syslog Proxy

[![Go Workflow][go_workflow_badge]][go_workflow]
[![Coverage Status][coverage_badge]][coverage]
[![Go Report][report_badge]][report]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]
[![Docker][docker_badge]][docker]

---

## Table of Contents

1. [Introduction](#introduction)
1. [Usage](#usage)
1. [Contributing](#contributing)
1. [License](#license)

## Introduction

_Axiom Syslog Proxy_ ships logs to Axiom, acting as a Syslog server.

## Installation

### Download the pre-compiled and archived binary manually

Binary releases are available on [GitHub Releases][2].

  [2]: https://github.com/axiomhq/axiom-syslog-proxy/releases/latest

### Install using [Homebrew](https://brew.sh)

```shell
brew tap axiomhq/tap
brew install axiom-syslog-proxy
```

To update:

```shell
brew update
brew upgrade axiom-syslog-proxy
```

### Install using `go get`

```shell
go get -u github.com/axiomhq/axiom-syslog-proxy/cmd/axiom-syslog-proxy
```

### Install from source

```shell
git clone https://github.com/axiomhq/axiom-syslog-proxy.git
cd axiom-syslog-proxy
make install
```

### Run the Docker image

Docker images are available on [DockerHub][docker].

## Usage

1. Set the following environment variables:

* `AXIOM_DEPLOYMENT_URL`: URL of the Axiom deployment to use
* `AXIOM_ACCESS_TOKEN`: **Personal Access** or **Ingest** token. Can be
  created under `Profile` or `Settings > Ingest Tokens`. For security reasons
  it is advised to use an Ingest Token with minimal privileges only.
* `AXIOM_INGEST_DATASET`: Dataset to ingest into

2. Run it: `./axiom-syslog-proxy` or using docker:

```shell
docker run -p601:601/tcp -p514:514/udp  \
  -e=AXIOM_DEPLOYMENT_URL=<AXIOM_DEPLOYMENT_URL> \
  -e=AXIOM_ACCESS_TOKEN=<AXIOM_ACCESS_TOKEN> \
  -e=AXIOM_INGEST_DATASET=<AXIOM_INGEST_DATASET> \
  axiomhq/axiom-syslog-proxy
```

3. Test it:

```shell
echo -n "tcp message" | nc -w1 localhost 601
echo -n "udp message" | nc -u -w1 localhost 514
```

## Contributing

Feel free to submit PRs or to fill issues. Every kind of help is appreciated. 

Before committing, `make` should run without any issues.

Kindly check our [Contributing](Contributing.md) guide on how to propose
bugfixes and improvements, and submitting pull requests to the project.

## License

&copy; Axiom, Inc., 2021

Distributed under MIT License (`The MIT License`).

See [LICENSE](LICENSE) for more information.

<!-- Badges -->

[go_workflow]: https://github.com/axiomhq/axiom-syslog-proxy/actions/workflows/push.yml
[go_workflow_badge]: https://img.shields.io/github/workflow/status/axiomhq/axiom-syslog-proxy/Push?style=flat-square&ghcache=unused
[coverage]: https://codecov.io/gh/axiomhq/axiom-syslog-proxy
[coverage_badge]: https://img.shields.io/codecov/c/github/axiomhq/axiom-syslog-proxy.svg?style=flat-square&ghcache=unused
[report]: https://goreportcard.com/report/github.com/axiomhq/axiom-syslog-proxy
[report_badge]: https://goreportcard.com/badge/github.com/axiomhq/axiom-syslog-proxy?style=flat-square&ghcache=unused
[release]: https://github.com/axiomhq/axiom-syslog-proxy/releases/latest
[release_badge]: https://img.shields.io/github/release/axiomhq/axiom-syslog-proxy.svg?style=flat-square&ghcache=unused
[license]: https://opensource.org/licenses/MIT
[license_badge]: https://img.shields.io/github/license/axiomhq/axiom-syslog-proxy.svg?color=blue&style=flat-square&ghcache=unused
[docker]: https://hub.docker.com/r/axiomhq/axiom-syslog-proxy
[docker_badge]: https://img.shields.io/docker/pulls/axiomhq/axiom-syslog-proxy.svg?style=flat-square&ghcache=unused
