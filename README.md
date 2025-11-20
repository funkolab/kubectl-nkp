# kubectl-nkp

A kubectl plugin to connect to NKP clusters using Cluster API.

## Features

- List NKP clusters
- Use kubeconfig secrets stored in the mgmt cluster to connect
- **Support multiple management kubeconfig files in `~/.kube/nkp/`** (user is prompted to select one if more than one is present)
- Connect to the selected cluster with two modes:
  - **Temporary connection**: Launch a shell with the corresponding kubeconfig (default behavior)
  - **Permanent connection**: Copy kubeconfig to `~/.kube/config` for persistent access

## Usage

1. Place one or more management cluster kubeconfig files in `~/.kube/nkp/`.
   - Each file should be a valid kubeconfig for a management cluster.
2. Run one of the following commands:

   **Temporary connection (default):**
   ```sh
   kubectl nkp connect
   ```
   This launches a temporary shell with `KUBECONFIG` set to the selected workload cluster's kubeconfig.

   **Permanent connection:**
   ```sh
   kubectl nkp connect -p
   # or
   kubectl nkp connect --permanent
   ```
   This copies the kubeconfig to `~/.kube/config`, making it the default kubeconfig for all `kubectl` commands.

3. If multiple kubeconfig files are present, you will be prompted to select one.
4. Select a Cluster API cluster from the list.
5. Depending on the mode:
   - **Temporary**: A shell will be launched with the workload cluster's kubeconfig
   - **Permanent**: The kubeconfig will be copied to `~/.kube/config` (existing config is backed up to `~/.kube/config.backup`)

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
