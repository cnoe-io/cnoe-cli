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
  verify      verify if the deployment exists
  version     Print the version number of Hugo

Flags:
  -c, --config string       path to config file (default "./config.yaml")
  -h, --help                help for cnoe
  -k, --kubeconfig string   path to the kubeconfig file (default "~/.kube/config")

Use "cnoe [command] --help" for more information about a command.
```

## Test

```bash
~ cd pkg/cmd
~ ginkgo run
```
