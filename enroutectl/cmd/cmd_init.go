// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Saaras Inc.
package cmd

import (
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"os"
)

type EnrouteCtlConfig struct {
	KCfg                 string `json:"kubeconfig,omitempty"`
	EnrouteCtlCfgFile    string `json:"enroutectlconfig,omitempty"`
	Verbose              bool   `json:"verbose,omitempty"`
	TargetStandalone     string `json:"targetstandalone,omitempty"`
	TargetStandaloneYaml string `json:"targetstandaloneyaml,omitempty"`
	TargetKSYaml         string `json:"targetkyaml,omitempty"`
	EnrouteCtlUUID       string `json:"enroutectluuid,omitempty"`
	OpenApiSpecFile      string `json:"openapispecfile,omitempty"`
}

var EnrouteCtlCfg EnrouteCtlConfig

const EnrouteCtlConfigFileName string = "enroutectl.yaml"

func enroutectl_initt(cmd *cobra.Command, args []string) {
	_, err := os.Stat(EnrouteCtlConfigFileName)
	if os.IsNotExist(err) {
		fmt.Printf(" Creating empty config file enroutectl.yaml \n")
		u, _ := uuid.NewRandom()
		EnrouteCtlCfg.EnrouteCtlUUID = u.String()
		ecfg := EnrouteCtlConfig{
			EnrouteCtlUUID: EnrouteCtlCfg.EnrouteCtlUUID,
		}
		ecfgmarshalled, _ := json.Marshal(ecfg)
		_ = ioutil.WriteFile(EnrouteCtlConfigFileName, ecfgmarshalled, 0644)
	} else {
		fmt.Printf(" enroutectl.yaml present, not creating \n")
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&EnrouteCtlCfg.EnrouteCtlCfgFile,
		"enroutectl-config",
		"",
		"config for enroutectl if no enroutectl.yaml (or enroutectl.json) found in current OR home directory ",
	)

	rootCmd.PersistentFlags().BoolVar(&EnrouteCtlCfg.Verbose,
		"verbose",
		false, //default
		"be verbose when running",
	)

	cobra.OnInitialize(InitConfig)
	addSubCommands()
}

func InitConfig() {
	foundcfgfile := false

	if EnrouteCtlCfg.EnrouteCtlCfgFile != "" {
		viper.SetConfigFile(EnrouteCtlCfg.EnrouteCtlCfgFile)
		foundcfgfile = true
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName("enroutectl")
	}

	// read in environment variables that match
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
		foundcfgfile = true
	}

	if !foundcfgfile {
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {

			enroutectl_initt(nil, nil)

			fmt.Printf(`

            ----------------------------------------------------
            Did not find enroutectl.yaml in current directory
            Wrote empty enroutectl.yaml
            ----------------------------------------------------

            `)
		} else {
			if EnrouteCtlCfg.Verbose {
				fmt.Println("Using config file:", viper.ConfigFileUsed())
			}
		}
	}
	err := viper.Unmarshal(&EnrouteCtlCfg)
	if err != nil {
		log.Fatalf("Error unable to decode config file into struct, %v", err)
	}
}
