# kubectl-nkp

`kubectl-nkp` is a CLI tool to connect easily to NKP workload clusters.

## Features

- List and select NKP workload clusters using fuzzy finder.
- Connect to selected cluster and launch a temporary shell with the corresponding kubeconfig.

## Usage

Your management cluster kubeconfig need to be put in the following path: `~/.kube/nkp/config`.

To use `kubectl-nkp`, run the following command:

```sh
kubectl nkp connect
```

This will list the available NKP workload clusters and allow you to select one using a fuzzy finder. Once selected, it will launch a shell with the kubeconfig for the selected cluster. If there is only one cluster available, it will connect to that cluster directly.

## Development

### Prerequisites

- Go 1.24.1 or later
- Kubernetes cluster with Cluster API installed

### Building

To build the project, run:

```sh
task build 
```

### Installing

To install the project, use:

```sh
task install
```

### Testing

To run linting, use:

```sh
task lint
```

## License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.