# CNOE CLI

The CNOE CLI is actively being developed to setup and configure the CNOE IDP for
its target users.

## Build

```
./hack/build.sh
```

## All Commands
```
cnoe cli for building your developer platform

Usage:
  cnoe [flags]
  cnoe [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  k8s         Run against a kubernets cluster
  template    Generate backstage templates from CRD/XRD
  version     Print the version number of Hugo

Flags:
  -h, --help   help for cnoe

Use "cnoe [command] --help" for more information about a command.


❯ ./cnoe k8s -h
Commands that assume a kubernetes cluster as the backend

Usage:
  cnoe k8s [command]

Available Commands:
  verify      Verify if the deployment exists

Flags:
  -h, --help                help for k8s
  -k, --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")

Use "cnoe k8s [command] --help" for more information about a command.
```

## Test

```bash
~ cd pkg/cmd
~ ginkgo run
```
