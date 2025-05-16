# kubectl-nkp

A kubectl plugin to connect to NKP workload clusters using Cluster API.

## Features

- List NKP clusters
- Use kubeconfig secrets stored in the mgmt cluster to connect
- **Support multiple management kubeconfig files in `~/.kube/nkp/`** (user is prompted to select one if more than one is present)
- Connect to the selected cluster and launch a temporary shell with the corresponding kubeconfig.

## Usage

1. Place one or more management cluster kubeconfig files in `~/.kube/nkp/`.
   - Each file should be a valid kubeconfig for a management cluster.
2. Run:
   ```sh
   kubectl nkp connect
   ```
3. If multiple kubeconfig files are present, you will be prompted to select one.
4. Select a Cluster API cluster from the list.
5. A shell will be launched with `KUBECONFIG` set to the selected workload cluster's kubeconfig.

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