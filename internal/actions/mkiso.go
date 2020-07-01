// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/logger"
	"github.com/lirios/ostree-image-creator/internal/oic"
)

type artifactsStruct struct {
	IsoFileName      string `yaml:"iso"`
	ChecksumFileName string `yaml:"checksum"`
}

// MkisoAction represents an action called "mkiso"
type MkisoAction struct {
	BaseAction    `yaml:",inline"`
	Label         string          `yaml:"label"`
	Volume        string          `yaml:"volume"`
	Path          string          `yaml:"path"`
	IsoLinux      bool            `yaml:"isolinux"`
	EfiBoot       bool            `yaml:"efiboot"`
	ImplantIsoMd5 bool            `yaml:"implantisomd5"`
	Artifacts     artifactsStruct `yaml:"artifacts"`
}

// Validate checks if the action is configured correctly
func (a *MkisoAction) Validate(context *oic.Context) error {
	if a.Label == "" {
		return fmt.Errorf("property \"label\" is mandatory for the \"%s\" action", a)
	} else if len(a.Label) == 32 {
		a.Label = a.Label[:32]
	}

	if a.Volume == "" {
		a.Volume = a.Label
	}

	if a.Path == "" {
		return fmt.Errorf("property \"path\" is mandatory for the \"%s\" action", a)
	}
	a.Path = filepath.Join(context.WorkspaceDir, a.Path)

	if a.Artifacts.IsoFileName == "" {
		return fmt.Errorf("property \"artifacts.iso\" is mandatory for the \"%s\" action", a)
	}
	a.Artifacts.IsoFileName = filepath.Join(a.Path, a.Artifacts.IsoFileName)

	if a.Artifacts.ChecksumFileName != "" {
		a.Artifacts.ChecksumFileName = filepath.Join(a.Path, a.Artifacts.ChecksumFileName)
	}

	return nil
}

// Run runs the action
func (a *MkisoAction) Run(context *oic.Context) error {
	if err := a.geniso(context); err != nil {
		return fmt.Errorf("failed to run genisoimage: %v", err)
	}

	if context.Architecture == "x86_64" {
		if err := a.isohybrid(context); err != nil {
			return fmt.Errorf("failed to run isohybrid: %v", err)
		}
	}

	if a.ImplantIsoMd5 {
		if err := a.implantisomd5(context); err != nil {
			return fmt.Errorf("failed to implant MD5: %v", err)
		}
	}

	if a.Artifacts.ChecksumFileName != "" {
		if err := a.sha256sum(context); err != nil {
			return fmt.Errorf("failed to calculate checksum: %v", err)
		}
	}

	// Move files to the output directory
	logger.Action("Moving ISO file to the output directory...")
	if err := executeCommand(context, "mv", a.Artifacts.IsoFileName, context.OutputDir); err != nil {
		return err
	}
	if a.Artifacts.ChecksumFileName != "" {
		logger.Action("Moving ISO checksum to the output directory...")
		if err := executeCommand(context, "mv", a.Artifacts.ChecksumFileName, context.OutputDir); err != nil {
			return err
		}
	}

	return nil
}

func (a *MkisoAction) geniso(context *oic.Context) error {
	args := []string{
		"genisoimage",
		"-verbose",
		"-V", a.Label,
		"-volset", a.Volume,
		"-rational-rock",
		"-J", "-joliet-long",
	}

	if a.IsoLinux {
		args = append(args,
			"-eltorito-boot", "isolinux/isolinux.bin",
			"-eltorito-catalog", "isolinux/boot.cat",
			"-no-emul-boot",
			"-boot-load-size", "4",
			"-boot-info-table")
	}

	if a.EfiBoot {
		args = append(args,
			"-eltorito-alt-boot",
			"-efi-boot", "images/efiboot.img",
			"-no-emul-boot")
	}

	additionalargs := []string{
		"-o", a.Artifacts.IsoFileName, ".",
	}
	args = append(args, additionalargs...)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = a.Path
	return oic.RunCommand(cmd)
}

func (a *MkisoAction) isohybrid(context *oic.Context) error {
	args := []string{
		"isohybrid",
		a.Artifacts.IsoFileName,
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = a.Path
	return oic.RunCommand(cmd)
}

func (a *MkisoAction) implantisomd5(context *oic.Context) error {
	args := []string{
		"implantisomd5",
		a.Artifacts.IsoFileName,
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = a.Path
	return oic.RunCommand(cmd)
}

func (a *MkisoAction) sha256sum(context *oic.Context) error {
	file, err := os.Create(a.Artifacts.ChecksumFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	args := []string{
		"sha256sum",
		"-b",
		"--tag",
		filepath.Base(a.Artifacts.IsoFileName),
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = filepath.Dir(a.Artifacts.ChecksumFileName)
	return oic.RunCommandWithOutput(cmd, file)
}
