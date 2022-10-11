package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/reproducible-containers/repro-get/pkg/cache"
	"github.com/reproducible-containers/repro-get/pkg/filespec"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newIPFSPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [flags] SHA256SUMS",
		Short: "Push the files into IPFS",
		Long: `Push the files into IPFS, and append the CIDs to the hash file.
Needs 'ipfs' command (https://github.com/ipfs/kubo) to be installed.

There is no 'repro-get ipfs pull' command.
To pull the pushed packages, set the provider to a {{.CID}} template string
with an IPFS gateway, such as:
$ repro-get --provider=http://ipfs.io/ipfs/{{.CID}} install
`,
		Example: "  repro-get ipfs push SHA256SUMS",
		Args:    cobra.ExactArgs(1),
		RunE:    ipfsPushAction,

		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.Bool("append", true, "Append the CIDs to the hash file")

	return cmd
}

func ipfsPushAction(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	flags := cmd.Flags()
	hashFile := args[0]
	cacheStr, err := flags.GetString("cache")
	if err != nil {
		return err
	}
	cache, err := cache.New(cacheStr)
	if err != nil {
		return err
	}

	fileSpecs, err := filespec.NewFromSHA256SUMSFiles(hashFile)
	if err != nil {
		return err
	}
	ipfsExe, err := exec.LookPath("ipfs")
	if err != nil {
		return err
	}

	appendFlag, err := flags.GetBool("append")
	if err != nil {
		return err
	}
	var appender io.WriteCloser
	if appendFlag {
		appender, err = os.OpenFile(hashFile, os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open %q with O_WRONLY|O_APPEND: %w", hashFile, err)
		}
		defer appender.Close()
	}

	for fname, fileSpec := range fileSpecs {
		if fileSpec.CID != "" {
			logrus.Infof("Skipping to push %q (Already has CID %q)", fname, fileSpec.CID)
			continue
		}
		blobPath, err := cache.BlobAbsPath(fileSpec.SHA256)
		if err != nil {
			return err
		}
		if _, err := os.Stat(blobPath); err != nil {
			return fmt.Errorf("uncached file? %q: %w (Hint: try 'repro-get download ...')", fname, err)
		}
		ipfsCmd := exec.CommandContext(ctx, ipfsExe, "add", "-Q", "--dereference-args", blobPath)
		ipfsCmd.Stderr = stderr
		logrus.Debugf("Running %v", ipfsCmd.Args)
		cidB, err := ipfsCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to execute %v: %w", ipfsCmd.Args, err)
		}
		cid := strings.TrimSpace(string(cidB))
		newLine := fmt.Sprintf("%s  /ipfs/%s", fileSpec.SHA256, cid)
		if _, err = fmt.Fprintln(stdout, newLine); err != nil {
			return err
		}
		if appender != nil {
			if _, err = fmt.Fprintln(appender, newLine); err != nil {
				return err
			}
		}
	}
	return nil
}
