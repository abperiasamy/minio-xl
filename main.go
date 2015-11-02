/*
 * Minio Cloud Storage, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/minio-xl/pkg/probe"
)

// minioConfig - http server config
type minioConfig struct {
	Address           string
	ControllerAddress string
	RPCAddress        string
	Anonymous         bool
	TLS               bool
	CertFile          string
	KeyFile           string
	RateLimit         int
}

func init() {
	// Check for the environment early on and gracefuly report.
	_, err := user.Current()
	if err != nil {
		Fatalf("Unable to obtain user's home directory. \nError: %s\n", err)
	}

	if os.Geteuid() == 0 {
		Fatalln("Please run ‘minio’ as a non-root user.")
	}

	// Check if minio was compiled using a supported version of Golang.
	checkGolangRuntimeVersion()
}

// Tries to get os/arch/platform specific information
// Returns a map of current os/arch/platform/memstats
func getSystemData() map[string]string {
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	mem := fmt.Sprintf("Used: %s | Allocated: %s | Used-Heap: %s | Allocated-Heap: %s",
		humanize.Bytes(memstats.Alloc),
		humanize.Bytes(memstats.TotalAlloc),
		humanize.Bytes(memstats.HeapAlloc),
		humanize.Bytes(memstats.HeapSys))
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s",
		host,
		runtime.GOOS,
		runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %s", runtime.Version(), strconv.Itoa(runtime.NumCPU()))
	return map[string]string{
		"PLATFORM": platform,
		"RUNTIME":  goruntime,
		"MEM":      mem,
	}
}

func findClosestCommands(command string) []string {
	var closestCommands []string
	for _, value := range commandsTree.PrefixMatch(command) {
		closestCommands = append(closestCommands, value.(string))
	}
	return closestCommands
}

func registerApp() *cli.App {
	// register all commands
	registerCommand(xlCmd)
	registerCommand(serverCmd)
	registerCommand(controllerCmd)
	registerCommand(versionCmd)

	// register all flags
	registerFlag(addressFlag)
	registerFlag(addressControllerFlag)
	registerFlag(addressServerRPCFlag)
	registerFlag(ratelimitFlag)
	registerFlag(anonymousFlag)
	registerFlag(certFlag)
	registerFlag(keyFlag)
	registerFlag(jsonFlag)

	// set up app
	app := cli.NewApp()
	app.Name = "minio-xl"
	// hide --version flag, version is a command
	app.HideVersion = true
	app.Author = "Minio.io"
	app.Usage = "Minio Cloud Storage"
	app.Description = `This version of the Minio binary is built using XL distribute erasure code backend. XL erasure codes each data block with - 8 Data x 8 Parity. XL is designed for immutable objects.`
	app.Flags = flags
	app.Commands = commands

	app.CustomAppHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

DESCRIPTION:
  {{.Description}}

USAGE:
  {{.Name}} {{if .Flags}}[global flags] {{end}}command{{if .Flags}} [command flags]{{end}} [arguments...]

COMMANDS:
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .Flags}}
GLOBAL FLAGS:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
VERSION:
  ` + minioXLVersion +
		`{{range $key, $value := ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}
`
	app.CommandNotFound = func(ctx *cli.Context, command string) {
		msg := fmt.Sprintf("‘%s’ is not a minio-xl sub-command. See ‘minio-xl help’.", command)
		closestCommands := findClosestCommands(command)
		if len(closestCommands) > 0 {
			msg += fmt.Sprintf("\n\nDid you mean one of these?\n")
			for _, cmd := range closestCommands {
				msg += fmt.Sprintf("        ‘%s’\n", cmd)
			}
		}
		Fatalln(msg)
	}

	return app
}

func main() {
	// Initialize probe.
	probe.Init()
	// Include release tag in the debug dumps.
	probe.SetAppInfo("Release-Tag", minioXLReleaseTag)
	probe.SetAppInfo("Commit-ID", minioXLShortCommitID)

	app := registerApp()
	app.Before = func(c *cli.Context) error {
		globalJSONFlag = c.GlobalBool("json")
		return nil
	}
	app.ExtraInfo = func() map[string]string {
		return getSystemData()
	}

	app.RunAndExitOnError()
}
