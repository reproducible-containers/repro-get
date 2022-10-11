package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/reproducible-containers/repro-get/pkg/archutil"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/reproducible-containers/repro-get/pkg/sha256sums"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newHashUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update SHA256SUMS",
		Short:   "Update the hash file",
		Example: "  repro-get hash update SHA256SUMS-" + archutil.OCIArchDashVariant(),
		Args:    cobra.ExactArgs(1),
		RunE:    hashUpdateAction,

		DisableFlagsInUseLine: true,
	}

	return cmd
}

func hashUpdateAction(cmd *cobra.Command, args []string) error {
	d, err := getDistro(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	hashFile := args[0]
	old, err := os.ReadFile(hashFile)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", hashFile, err)
	}
	sums, err := sha256sums.Parse(bytes.NewReader(old))
	if err != nil {
		return fmt.Errorf("failed to parse %q as SHA256SUMS: %w", hashFile, err)
	}
	fileSpecs, err := filespec.NewFromSHA256SUMS(sums)
	if err != nil {
		return err
	}

	var pkgs []string
	for _, f := range fileSpecs {
		pkg, err := d.PackageName(*f)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to resolve the package name of %q", f.Name)
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	opts := distro.HashOpts{
		FilterByName: pkgs,
	}
	var b bytes.Buffer
	hw := distro.NewHashWriter(&b)
	if err := d.GenerateHash(ctx, hw, opts); err != nil {
		return err
	}
	if b.Len() == 0 {
		return errors.New("no hash was generated")
	}
	neu := b.Bytes()
	if bytes.Equal(old, neu) {
		logrus.Info("No update")
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), cmp.Diff(string(old), string(neu)))
	return os.WriteFile(hashFile, neu, 0644)
}
