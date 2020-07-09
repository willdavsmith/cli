// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/dapr/cli/pkg/kubernetes"
	"github.com/dapr/cli/pkg/print"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

const (
	// dashboardSvc is the name of the dashboard service running in cluster
	dashboardSvc = "dapr-dashboard"

	// defaultHost is the default host used for port forwarding for `dapr dashboard`
	defaultHost = "localhost"

	// defaultLocalPort is the default local port used for port forwarding for `dapr dashboard`
	defaultLocalPort = 8080

	// defaultNamespace is the default namespace where the dashboard is deployed
	defaultNamespace = "dapr-system"

	// remotePort is the port dapr dashboard pod is listening on
	remotePort = 8080
)

var dashboardNamespace string
var localPort int

var DashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start Dapr dashboard",
	Run: func(cmd *cobra.Command, args []string) {
		if port < 0 {
			localPort = defaultLocalPort
		} else {
			localPort = port
		}

		config, client, err := kubernetes.GetKubeConfigClient()
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to initialize kubernetes client")
			os.Exit(1)
		}

		// manage termination of port forwarding connection on interrupt
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		defer signal.Stop(signals)

		// search for dashboard service namespace in order:
		// dapr-system, default
		foundNamespace := ""
		namespaces := []string{"dapr-system", "default"}
		for _, namespace := range namespaces {
			ok := kubernetes.CheckPodExists(client, namespace, nil, dashboardSvc)
			if ok {
				foundNamespace = namespace
				break
			}
		}

		// if the service is not found, error out, tell user to supply a namespace
		if foundNamespace == "" {
			print.FailureStatusEvent(os.Stdout, "Failed to find Dapr dashboard in namespaces: %v\nIf Dapr dashboard is deployed to a different namespace, please use dapr dashboard -n", namespaces)
			os.Exit(1)
		}

		portForward, err := kubernetes.NewPortForward(
			config,
			foundNamespace,
			dashboardSvc,
			defaultHost,
			localPort,
			remotePort,
			false,
		)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "%s\n", err)
			os.Exit(1)
		}

		// initialize port forwarding
		if err = portForward.Init(); err != nil {
			print.FailureStatusEvent(os.Stdout, "Error in port forwarding: %s\nCheck for `dapr dashboard` running in other terminal sessions, or use the `--port` flag to use a different port.\n", err)
			os.Exit(1)
		}

		// block until interrupt signal is received
		go func() {
			<-signals
			portForward.Stop()
		}()

		// url for dashboard after port forwarding
		var webURL string = fmt.Sprintf("http://%s:%d", defaultHost, localPort)

		print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard found in namespace:\t%s\n", foundNamespace))
		print.InfoStatusEvent(os.Stdout, fmt.Sprintf("Dapr dashboard available at:\t%s\n", webURL))

		err = browser.OpenURL(webURL)
		if err != nil {
			print.FailureStatusEvent(os.Stdout, "Failed to start Dapr dashboard in browser automatically")
			print.FailureStatusEvent(os.Stdout, fmt.Sprintf("Visit %s in your browser to view the dashboard", webURL))
		}

		<-portForward.GetStop()
	},
}

func init() {
	DashboardCmd.Flags().BoolVarP(&kubernetesMode, "kubernetes", "k", false, "Start Dapr dashboard in local browser")
	DashboardCmd.Flags().IntVarP(&port, "port", "p", defaultLocalPort, "The local port on which to serve dashboard")
	DashboardCmd.MarkFlagRequired("kubernetes")
	RootCmd.AddCommand(DashboardCmd)
}
