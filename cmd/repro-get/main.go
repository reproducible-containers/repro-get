package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/distro/alpine"
	"github.com/reproducible-containers/repro-get/pkg/distro/arch"
	"github.com/reproducible-containers/repro-get/pkg/distro/debian"
	"github.com/reproducible-containers/repro-get/pkg/distro/distroutil/detect"
	"github.com/reproducible-containers/repro-get/pkg/distro/fedora"
	"github.com/reproducible-containers/repro-get/pkg/distro/none"
	"github.com/reproducible-containers/repro-get/pkg/distro/ubuntu"
	"github.com/reproducible-containers/repro-get/pkg/envutil"
	"github.com/reproducible-containers/repro-get/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}

var knownDistros = map[string]distro.Distro{
	none.Name:   none.New(),
	debian.Name: debian.New(),
	ubuntu.Name: ubuntu.New(),
	fedora.Name: fedora.New(),
	alpine.Name: alpine.New(),
	arch.Name:   arch.New(),
}

func knownDistroNames() []string {
	var ss []string
	for k := range knownDistros {
		ss = append(ss, k)
	}
	sort.Strings(ss)
	return ss
}

func getDistroByName(name string) (distro.Distro, error) {
	if name == "" {
		detected := detect.DistroID()
		if _, ok := knownDistros[detected]; ok {
			name = detected
		} else {
			logrus.Debugf("Unsupported distro %q", detected)
			name = none.Name
		}
	}
	if d, ok := knownDistros[name]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("unknown distro %q (known distros: %v)", name, knownDistroNames())
}

func getDistro(cmd *cobra.Command) (distro.Distro, error) {
	name, err := cmd.Flags().GetString("distro")
	if err != nil {
		return nil, err
	}
	d, err := getDistroByName(name)
	if err != nil {
		return nil, err
	}
	info := d.Info()
	logrus.Debugf("Using distro driver %q", info.Name)
	if info.Experimental {
		logrus.Warnf("Distro driver %q is experimental", info.Name)
	}
	return d, nil
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repro-get",
		Short: "Reproducible apt, dnf, apk, and pacman",
		Example: `  Generate the hash file for all the installed packages:
  $ repro-get hash generate >SHA256SUMS

  Install packages using the hash file:
  $ repro-get install SHA256SUMS
`,
		Version:       strings.TrimPrefix(version.GetVersion(), "v"),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	flags := cmd.PersistentFlags()
	flags.Bool("debug", envutil.Bool("DEBUG", false), "debug mode [$DEBUG]")
	flags.String("cache", envutil.String("REPRO_GET_CACHE", "/var/cache/repro-get"), "Cache directory [$REPRO_GET_CACHE]")

	defaultDistro, err := getDistroByName("")
	if err != nil {
		panic(err)
	}

	flags.String("distro", envutil.String("REPRO_GET_DISTRO", defaultDistro.Info().Name), "Distribution driver [$REPRO_GET_DISTRO]")
	_ = cmd.RegisterFlagCompletionFunc("distro", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return knownDistroNames(), cobra.ShellCompDirectiveNoFileComp
	})
	// the actual default value is filled after resolving the distro
	flags.StringSlice("provider", envutil.StringSlice("REPRO_GET_PROVIDER", nil), "File provider, run 'repro-get info' to show the default [$REPRO_GET_PROVIDER]")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}

	cmd.AddCommand(
		newInfoCommand(),
		newInstallCommand(),
		newDownloadCommand(),
		newHashCommand(),
		newCacheCommand(),
		newIPFSCommand(),
		newDockerfileCommand(),
	)
	return cmd
}

func needsSubcommand(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}
