/*
Copyright © 2025 Christophe Jauffret <reg-github@geo6.net>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterItem represents a cluster for fuzzy finding
type ClusterItem struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	CreatedAt   string
}

// ClusterAPI resource definitions
var clusterGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta1",
	Resource: "clusters",
}

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a NKP workload cluster",
	Long:  `Connect to a NKP workload cluster by selecting a cluster and using the kubeconfig stored in a secret.`,
	Run: func(cmd *cobra.Command, args []string) {
		permanent, _ := cmd.Flags().GetBool("permanent")
		// Path to the kubeconfig directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			cobra.CheckErr(err)
		}

		nkpKubeconfigDir := filepath.Join(homeDir, ".kube", "nkp")

		// List kubeconfig files in the directory
		files, err := os.ReadDir(nkpKubeconfigDir)
		if err != nil {
			cobra.CheckErr(err)
		}

		var kubeconfigFiles []string
		for _, file := range files {
			if !file.IsDir() {
				kubeconfigFiles = append(kubeconfigFiles, filepath.Join(nkpKubeconfigDir, file.Name()))
			}
		}

		if len(kubeconfigFiles) == 0 {
			cobra.CheckErr(fmt.Errorf("No kubeconfig files found in %s", nkpKubeconfigDir))
		}

		// Sort kubeconfigFiles for predictable order
		sort.Strings(kubeconfigFiles)

		var selectedKubeconfig string
		if len(kubeconfigFiles) == 1 {
			selectedKubeconfig = kubeconfigFiles[0]
			fmt.Printf("Using kubeconfig: %s\n", filepath.Base(selectedKubeconfig))
		} else {
			idx, err := fuzzyfinder.Find(
				kubeconfigFiles,
				func(i int) string {
					return filepath.Base(kubeconfigFiles[i])
				},
				fuzzyfinder.WithPromptString("Select management kubeconfig > "),
			)
			if err != nil {
				cobra.CheckErr(err)
			}
			selectedKubeconfig = kubeconfigFiles[idx]
			fmt.Printf("Using kubeconfig: %s\n", filepath.Base(selectedKubeconfig))
		}

		// Build config from selected kubeconfig file
		config, err := clientcmd.BuildConfigFromFlags("", selectedKubeconfig)
		if err != nil {
			fmt.Printf("Error building kubeconfig: %v\n", err)
			cobra.CheckErr(err)
		}

		// Create dynamic client for ClusterAPI resources
		dynamicClient, err := dynamic.NewForConfig(config)
		if err != nil {
			fmt.Printf("Error creating dynamic client: %v\n", err)
			cobra.CheckErr(err)
		}

		// Create client for core resources (secrets)
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			fmt.Printf("Error creating kubernetes client: %v\n", err)
			cobra.CheckErr(err)
		}

		ctx := context.Background()
		var clusterList *unstructured.UnstructuredList
		var listErr error

		clusterList, listErr = dynamicClient.Resource(clusterGVR).List(ctx, metav1.ListOptions{})

		if listErr != nil {
			fmt.Printf("Error listing Cluster API clusters: %v\n", listErr)
			cobra.CheckErr(listErr)
		}

		var filteredClusterItems []unstructured.Unstructured
		// for _, item := range clusterList.Items {
		// 	// Skip clusters in the default namespace
		// 	if item.GetNamespace() != "default" {
		// 		filteredClusterItems = append(filteredClusterItems, item)
		// 	}
		// }
		filteredClusterItems = append(filteredClusterItems, clusterList.Items...)

		if len(filteredClusterItems) == 0 {
			cobra.CheckErr(fmt.Errorf("no CAPI clusters found in the kubernetes cluster"))
		}

		clusterItems := filteredClusterItems
		var selectedCluster unstructured.Unstructured
		var idx int

		if len(filteredClusterItems) == 1 {
			selectedCluster = filteredClusterItems[0]
		} else {

			// Use fuzzy finder to select a cluster
			idx, err = fuzzyfinder.Find(
				clusterItems,
				func(i int) string {
					return clusterItems[i].GetName()
				},
				fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
					if i == -1 {
						return ""
					}

					creationTime := clusterItems[i].GetCreationTimestamp()

					// Build a preview with available cluster information
					preview := fmt.Sprintf("Name: %s\nNamespace: %s\nCreated: %s\n",
						clusterItems[i].GetName(),
						clusterItems[i].GetNamespace(),
						formatAge(creationTime))

					return preview
				}),
			)

			if err != nil {
				cobra.CheckErr(err)
			}
		}

		selectedCluster = clusterItems[idx]
		fmt.Printf("Connecting on cluster %s (namespace: %s)\n", selectedCluster.GetName(), selectedCluster.GetNamespace())

		// Use the namespace from the selected cluster
		clusterNamespace := selectedCluster.GetNamespace()
		clusterName := selectedCluster.GetName()
		secretName := fmt.Sprintf("%s-kubeconfig", clusterName)

		// Look for the kubeconfig secret in the same namespace as the cluster

		secret, err := clientset.CoreV1().Secrets(clusterNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err == nil {

			// Extract kubeconfig data
			kubeconfigData, ok := secret.Data["value"]
			if !ok {
				cobra.CheckErr(fmt.Errorf("Secret '%s' does not contain kubeconfig data under 'value' key", secretName))
			}

			// Use the kubeconfig data based on the permanent flag
			if permanent {
				copyKubeconfigToPermanentLocation(kubeconfigData, clusterName)
			} else {
				useKubeconfigData(kubeconfigData)
			}
			return
		} else {
			cobra.CheckErr(fmt.Errorf("Could not find secret '%s' in namespace '%s': %v", secretName, clusterNamespace, err))
		}

	},
}

// copyKubeconfigToPermanentLocation copies the kubeconfig data to ~/.kube/config
func copyKubeconfigToPermanentLocation(kubeconfigData []byte, clusterName string) {
	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		cobra.CheckErr(err)
	}

	kubeDir := filepath.Join(homeDir, ".kube")
	kubeconfigPath := filepath.Join(kubeDir, "config")

	// Create .kube directory if it doesn't exist
	if err := os.MkdirAll(kubeDir, 0755); err != nil {
		fmt.Printf("Error creating .kube directory: %v\n", err)
		cobra.CheckErr(err)
	}

	// Check if ~/.kube/config already exists and create backup
	if _, err := os.Stat(kubeconfigPath); err == nil {
		backupPath := kubeconfigPath + ".backup"
		if err := copyFile(kubeconfigPath, backupPath); err != nil {
			fmt.Printf("Error creating backup of existing kubeconfig: %v\n", err)
			cobra.CheckErr(err)
		}
		fmt.Printf("Existing kubeconfig backed up to: %s\n", backupPath)
	}

	// Write the new kubeconfig to ~/.kube/config
	if err := os.WriteFile(kubeconfigPath, kubeconfigData, 0600); err != nil {
		fmt.Printf("Error writing kubeconfig to %s: %v\n", kubeconfigPath, err)
		cobra.CheckErr(err)
	}

	fmt.Printf("✅ Kubeconfig for cluster '%s' has been written to %s\n", clusterName, kubeconfigPath)
	fmt.Printf("You can now use kubectl commands directly without any additional setup.\n")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return err
	}

	return nil
}

// useKubeconfigData creates a temporary file with the provided kubeconfig data
// and launches a shell with KUBECONFIG pointing to that file
func useKubeconfigData(kubeconfigData []byte) {
	// Create a temporary file for the kubeconfig
	tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		fmt.Printf("Error creating temporary file: %v\n", err)
		cobra.CheckErr(err)
	}
	defer tempFile.Close()

	// Write the kubeconfig to the temporary file
	if _, err := tempFile.Write(kubeconfigData); err != nil {
		fmt.Printf("Error writing to temporary file: %v\n", err)
		cobra.CheckErr(err)
	}

	// Launch a new shell with KUBECONFIG set to the temporary file
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	execCmd := exec.Command(shell)
	execCmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", tempFile.Name()))
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	fmt.Printf("Launching temporary shell\n")
	if err := execCmd.Run(); err != nil {
		fmt.Printf("Error running shell: %v\n", err)
	}

	fmt.Println("Shell session ended. Temporary kubeconfig will be cleaned up.")
	// Clean up the temporary file
	os.Remove(tempFile.Name())
}

func init() {
	rootCmd.AddCommand(connectCmd)

	// Add the permanent flag
	connectCmd.Flags().BoolP("permanent", "p", false, "Copy kubeconfig to ~/.kube/config instead of launching a temporary shell")
}

// formatAge formats the age of a resource similar to kubectl
func formatAge(creationTime metav1.Time) string {
	now := time.Now()
	diff := now.Sub(creationTime.Time)

	// Simple formatting for demonstration purposes
	seconds := int(diff.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	if days < 30 {
		return fmt.Sprintf("%dd", days)
	}
	months := days / 30
	return fmt.Sprintf("%dM", months)
}
