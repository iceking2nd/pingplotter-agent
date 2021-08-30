/*
Copyright © 2021 Daniel Wu <iceking2nd@gmail.com>

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
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//var cfgFile string
const SCRIPT_VERSION = "0.5"

var (
	addr string
	port int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pingplotter-agent",
	Short: "PingPlotter Agent",
	/*Long: `A longer description that spans multiple lines and likely contains
	examples and usage of using your application. For example:

	Cobra is a CLI library for Go that empowers applications.
	This application is a tool to generate the needed files
	to quickly create a Cobra application.`,*/
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		gin.SetMode(gin.ReleaseMode)
		r := gin.Default()
		r.Any("/*path", handlerPath)
		r.Run(fmt.Sprintf("%s:%d", addr, port))
	},
}

func handlerPath(ctx *gin.Context) {
	var (
		packetTypeMsg     string
		osName            string
		osVersion         string
		timeoutTimeString string
		timeoutTime       float64
		tracerouteCommand = "/usr/bin/traceroute -m 20 -n -q 1"
		uts               = &unix.Utsname{}
	)

	timeoutTimeString = ctx.DefaultQuery("TimeoutTime", "5000")
	timeoutTime, _ = strconv.ParseFloat(timeoutTimeString, 64)
	tracerouteCommand += fmt.Sprintf(" -w %.3f", func() float64 {
		if (timeoutTime-50)/1000 < 0 {
			return 0
		}
		return (timeoutTime - 50) / 1000
	}())

	_ = unix.Uname(uts)
	osName = string(bytes.Trim(uts.Sysname[:], "\x00"))
	osVersion = string(bytes.Trim(uts.Version[:], "\x00"))

	switch ctx.Query("PacketType") {
	case "TCP":
		packetTypeMsg = "Using TCP packets"
		tracerouteCommand += " -T"
		tcpPort, hasTCPPort := ctx.GetQuery("TCPPort")
		if hasTCPPort {
			tracerouteCommand += fmt.Sprintf(" -p %s", tcpPort)
			packetTypeMsg += fmt.Sprintf(" on port %s", tcpPort)
		} else {
			packetTypeMsg += " on default port"
		}
	case "UDP":
		packetTypeMsg = "Using UDP packets"
	default:
		tracerouteCommand += " -I"
		packetTypeMsg = "Using ICMP packets"
	}

	tracerouteCommand += " "
	tracerouteCommand += ctx.Query("IP")
	tracerouteCommand += " "
	tracerouteCommand += ctx.DefaultQuery("PacketSize", "56")

	ctx.Header("Content-type", "text/html")
	ctx.String(200, `<html>
<head>
    <title>PingPlotter remote trace agent, V%s</title>
    <!-- OS: %s %s -->
</head>
<body>
<p>PingPlotter remote trace agent, V%s<br>
    %s</p>
<hr><PRE>%s
</PRE><hr></body>
</html>`, SCRIPT_VERSION, osName, osVersion, SCRIPT_VERSION, packetTypeMsg, func(c string) string {
		_, ipExist := ctx.GetQuery("IP")
		if !ipExist {
			return "Not a valid IP address!"
		}
		var cmdRun *exec.Cmd
		if os.Geteuid() != 0 {
			cmdRun = exec.Command("sudo", strings.Split(c, " ")...)
		} else {
			cmdRun = exec.Command(strings.Split(c, " ")[0], strings.Split(c, " ")[1:]...)
		}

		cmdResult, err := cmdRun.Output()
		if err != nil {
			return err.Error()
		}
		return string(cmdResult)
	}(tracerouteCommand))
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	//rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pingplotter-agent.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 7465, "Listen port")
	rootCmd.PersistentFlags().StringVarP(&addr, "address", "l", "", "Listen address")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	/*if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".pingplotter-agent" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".pingplotter-agent")
	}*/

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	/*if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}*/
}
