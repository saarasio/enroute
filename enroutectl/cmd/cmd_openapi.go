// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package cmd

import (
	"fmt"
	"github.com/saarasio/enroute/enroutectl/config"
	"github.com/saarasio/enroute/enroutectl/openapi"
	"github.com/spf13/cobra"
	"os"
	//"github.com/davecgh/go-spew/spew"
)

func CheckErr(cmd *cobra.Command, e error, mesg string) {
	if e != nil {
		fmt.Printf("\n%s %s\n", mesg, e)
		os.Exit(1)
	}
}

func enroutectl_openapi(cmd *cobra.Command, args []string) {
	if EnrouteCtlCfg.OpenApiSpecFile == "" {
		fmt.Println("\nOpen API spec file not found. Please specify file using --openapi-spec flag")
		os.Exit(1)
	}

	_, err := os.Stat(EnrouteCtlCfg.OpenApiSpecFile)
	CheckErr(cmd, err, "Open API spec file not found. Please specify file using --openapi-spec flag")

	s, err3 := openapi.Spec(EnrouteCtlCfg.OpenApiSpecFile)
	CheckErr(cmd, err3, "Failed to parse openapi spec")

	ecfg := config.EnrouteConfig{
		EnrouteCtlUUID: EnrouteCtlCfg.EnrouteCtlUUID,
	}

	openapi.SwaggerToEnroute(s, &ecfg)

	// If no target specifed, assume verbose

	if EnrouteCtlCfg.Verbose ||
		(EnrouteCtlCfg.TargetStandalone == "" &&
			EnrouteCtlCfg.TargetStandaloneYaml == "" &&
			EnrouteCtlCfg.TargetKSYaml == "") {
		yamlout := openapi.WriteYAML(&ecfg)
		fmt.Printf("%s\n", yamlout)
	}

	if EnrouteCtlCfg.TargetStandalone != "" {
		// TODO: Validate URL
		openapi.CreateOnEnroute(EnrouteCtlCfg.TargetStandalone, &ecfg)
	}

	if EnrouteCtlCfg.TargetKSYaml != "" {
		openapi.CreateKSYaml(EnrouteCtlCfg.TargetKSYaml, &ecfg)
	}
}
