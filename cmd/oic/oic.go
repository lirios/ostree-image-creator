// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/lirios/ostree-image-creator/internal/actions"
	"github.com/lirios/ostree-image-creator/internal/logger"
	"github.com/lirios/ostree-image-creator/internal/oic"
)

func setupCloseHandler(context *oic.Context) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if !context.Keep {
			os.RemoveAll(context.WorkspaceDir)
		}
		os.Exit(0)
	}()
}

func run(manifest *actions.Manifest, context *oic.Context) int {
	for _, action := range manifest.Actions {
		logger.Actionf("Running: %s", action.Description())
		err := action.Run(context)
		defer action.Cleanup(context)
		if err != nil {
			logger.Errorf("Action \"%s\" failed: %v", action.Description(), err)
			return 1
		}
	}

	return 0
}

func main() {
	// Exit code
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	// Logger
	log.SetFlags(0)

	// Architecture (according to https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63)
	architectures := map[string]string{
		"386":         "x86",
		"amd64":       "x86_64",
		"amd64p32":    "x86",
		"arm":         "armhfp",
		"armbe":       "armhfp",
		"arm64":       "aarch64",
		"arm64be":     "aarch64",
		"ppc":         "ppc",
		"ppc64":       "ppc64",
		"ppc64le":     "ppc64",
		"mips":        "mips",
		"mipsle":      "mipsle",
		"mips64":      "mips64",
		"mips64le":    "mips64le",
		"mips64p32":   "mips64p32",
		"mips64p32le": "mips64p32le",
		"s390":        "s390",
		"s390x":       "s390x",
		"sparc":       "sparc",
		"sparc64":     "sparc64",
	}
	currentArch := architectures[runtime.GOARCH]

	var (
		arch      string
		outDir    string
		workspace string
		keep      bool
		force     bool
		verbose   bool
	)

	// Create template vars
	templateVars := make(map[string]interface{})
	templateVars["today"] = time.Now().Format("20060102")
	templateVars["now"] = time.Now().Format("15:04:05")

	// Build command
	var cmdBuild = &cobra.Command{
		Use:   "build [path to manifest file]",
		Short: "Build an image as specified in the manifest file",
		Long:  "build creates an image executing the actions from the manifest file",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Toggle debug output
			logger.SetVerbose(verbose)

			// Check if we have the manifest path
			if len(args) == 0 {
				logger.Error("Please specify the path of your manifest files")
				exitCode = 1
				return
			}

			// Absolute path to the manifest file
			manifestPath, err := filepath.Abs(args[0])
			if err != nil {
				logger.Errorf("Failed to get absolute representation of manifest path: %v", err)
				exitCode = 1
				return
			}

			outDir, err := filepath.Abs(outDir)
			if err != nil {
				logger.Errorf("Failed to get absolute representation of the output directory: %v", err)
				exitCode = 1
				return
			}

			// Absolute path to working directory
			absWorkspace, err := filepath.Abs(workspace)
			if err != nil {
				logger.Errorf("Failed to get absolute representation of working directory: %v", err)
				exitCode = 1
				return
			}

			// Create workspace directory
			if err := os.MkdirAll(absWorkspace, 0755); err != nil {
				logger.Errorf("Failed to create workspace directory: %v", err)
				exitCode = 1
				return
			}
			if keep {
				logger.Infof("Workspace directory: %s", absWorkspace)
			}

			// Template variables
			templateVars["architecture"] = arch

			// Open and parse the manifest
			logger.Actionf("Reading manifest file \"%s\"...", manifestPath)
			manifest, err := actions.OpenManifest(manifestPath, templateVars)
			if err != nil {
				logger.Errorf("Failed to open manifest \"%s\": %v", manifestPath, err)
				exitCode = 1
				return
			}

			// Append manifest variables to be used later by templates
			templateVars["kernelArguments"] = manifest.Variables.KernelArguments

			// Set current working directory to the location of the
			// manifest file, so that it can reference relative files
			manifestDir := filepath.Dir(manifestPath)
			if err := os.Chdir(manifestDir); err != nil {
				logger.Errorf("Failed to change current working directory: %v", err)
				exitCode = 1
				return
			}

			// Create a temporary directory for the output files
			tempDir, err := ioutil.TempDir(absWorkspace, "oic-")
			if err != nil {
				logger.Errorf("Failed to create temporary directory: %v", err)
				exitCode = 1
				return
			}
			if !keep {
				defer func() {
					if err := os.RemoveAll(tempDir); err != nil {
						logger.Errorf("Failed to delete temporary directory \"%s\": %v", tempDir, err)
					}
				}()
			}

			// Create a directory for various stuff
			scrapDir := filepath.Join(tempDir, "scrap")
			if err := os.MkdirAll(scrapDir, 0755); err != nil {
				logger.Errorf("Failed to create downloads directory: %v", err)
				exitCode = 1
				return
			}

			// Create a directory for downloads
			downloadDir := filepath.Join(tempDir, "downloads")
			if err := os.MkdirAll(downloadDir, 0755); err != nil {
				logger.Errorf("Failed to create downloads directory: %v", err)
				exitCode = 1
				return
			}

			// Context
			context := &oic.Context{
				Architecture: arch,
				ManifestDir:  manifestDir,
				OutputDir:    outDir,
				WorkspaceDir: tempDir,
				ScrapDir:     scrapDir,
				DownloadDir:  downloadDir,
				Keep:         keep,
				Force:        force,
				Verbose:      verbose,
				TemplateVars: templateVars,
			}

			// Exit gracefully
			setupCloseHandler(context)

			// Verify all actions
			for _, action := range manifest.Actions {
				err = action.Validate(context)
				if err != nil {
					logger.Errorf("Validation failed: %v", err)
					exitCode = 1
					return
				}
			}

			// Execute all the actions in order
			exitCode = run(manifest, context)
		},
	}
	cmdBuild.Flags().StringVarP(&arch, "arch", "a", currentArch, "architecture")
	cmdBuild.Flags().StringVarP(&outDir, "output", "o", ".", "path where all the artifacts go")
	cmdBuild.Flags().StringVarP(&workspace, "workspace", "w", "/var/tmp", "path where all the build files go")
	cmdBuild.Flags().BoolVarP(&keep, "keep", "k", false, "keep all the artifacts produced during the build")
	cmdBuild.Flags().BoolVarP(&force, "force", "f", false, "overwrite previously generated files")
	cmdBuild.Flags().BoolVarP(&verbose, "verbose", "v", false, "more messages during the build")

	// Resolve command
	var cmdResolve = &cobra.Command{
		Use:   "resolve [path to manifest file]",
		Short: "Parse the manifest file, expand all the expressions and print the text",
		Long:  "resolve parses the manifest file and print the expanded text",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Check if we have the manifest path
			if len(args) == 0 {
				logger.Error("Please specify the path of your manifest files")
				exitCode = 1
				return
			}

			// Template variables
			templateVars["architecture"] = arch

			// Manifest path
			manifestPath := args[0]

			// Open and parse the manifest
			logger.Actionf("Reading manifest file \"%s\"...", manifestPath)
			manifest, err := actions.OpenManifest(manifestPath, templateVars)
			if err != nil {
				logger.Errorf("Failed to open manifest \"%s\": %v", manifestPath, err)
				exitCode = 1
				return
			}

			// Print the manifest and exit
			fmt.Println(manifest)
		},
	}
	cmdResolve.Flags().StringVarP(&arch, "arch", "a", currentArch, "architecture")

	// Root command
	var rootCmd = &cobra.Command{Use: "oic"}
	rootCmd.AddCommand(cmdResolve)
	rootCmd.AddCommand(cmdBuild)
	rootCmd.Execute()
}
