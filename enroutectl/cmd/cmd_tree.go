// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "enroutectl",
	Short: "Utility to configure Enroute Standalone Gateway and Enroute Ingress Gateway",
	Long: `
    enroutectl can be used to generate declarative configuration for

        - Enroute Standalone Gateway
        - Kubernetes Ingress Gateway

    enroutectl can also read openapi spec to generate config for gateway

`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func enroutectl_empty(cmd *cobra.Command, args []string) {
	cmd.Help()
	os.Exit(0)
}

var rootInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize enroutectl",
	Long: `

    Initialize enroutectl. Writes enroutectl.yaml in current file if absent. It's a no-op if file present.

    Example:

    enroutectl init

    `,
	Run: enroutectl_initt,
}

var rootOpenApiCmd = &cobra.Command{
	Use:   "openapi",
	Short: "Automatically program Enroute Universal API Gateway using OpenAPI Spec",
	Long: `

    Program Enroute Universal API Gateway using an OpenAPI spec.
    This command takes open api spec file as input.

    It can be used to program a standalone gateway, a kubernetes ingress gateway or
    write a file with enroute declarative configuration

    Example:

    enroutectl openapi --openapi-spec petstore.json --to-standalone-url <http://localhost:1323/>
    enroutectl openapi --openapi-spec petstore.json --to-standalone-yaml <yaml-file-name>
    enroutectl openapi --openapi-spec petstore.json --to-k8s-yaml <yaml-file-name>
    enroutectl openapi --openapi-spec petstore.json --verbose

    `,
	Run: enroutectl_openapi,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func addSubCommands() {
	rootCmd.AddCommand(rootInitCmd)
	rootOpenApiCmd.Flags().StringVar(&EnrouteCtlCfg.TargetStandalone,
		"to-standalone-url",
		"",
		"url to running instance of standalone enroute gateway",
	)
	rootOpenApiCmd.Flags().StringVar(&EnrouteCtlCfg.TargetStandaloneYaml,
		"to-standalone-yaml",
		"",
		"name of yaml file to write configuration for Standalone Enroute Gateway",
	)
	rootOpenApiCmd.Flags().StringVar(&EnrouteCtlCfg.TargetKSYaml,
		"to-k8s-yaml",
		"",
		"name of yaml file to write configuration for Kubernetes Ingress Enroute Gateway",
	)
	rootOpenApiCmd.Flags().StringVar(&EnrouteCtlCfg.OpenApiSpecFile,
		"openapi-spec",
		"",
		"path to open api spec",
	)
	rootCmd.AddCommand(rootOpenApiCmd)
}
