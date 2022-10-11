package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/urlopener"
	"github.com/reproducible-containers/repro-get/pkg/version"
	"github.com/spf13/cobra"
)

func newInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "info",
		Short:   "Show diagnostic information",
		Example: "  repro-get info",
		Args:    cobra.NoArgs,
		RunE:    infoAction,

		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.Bool("json", false, "Enable JSON output")

	return cmd
}

func infoAction(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()
	jsonFlag, err := flags.GetBool("json")
	if err != nil {
		return err
	}
	info, err := GetInfo(cmd)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	if jsonFlag {
		b, err := json.MarshalIndent(info, "", "    ")
		if err != nil {
			return err
		}
		if _, err = fmt.Fprintln(w, string(b)); err != nil {
			return err
		}
		return nil
	}
	fmt.Fprintln(w, "Version: "+info.Version)
	fmt.Fprintln(w, "Cache: "+info.Cache)
	fmt.Fprintln(w, "Recognized schemes: "+strings.Join(info.Schemes, " "))
	fmt.Fprintln(w, "Recognized distros: "+strings.Join(info.Distros, " "))
	fmt.Fprintln(w, "Distro: "+info.Distro.Name)
	fmt.Fprintln(w, "Default providers:")
	for _, f := range info.Distro.DefaultProviders {
		fmt.Fprintln(w, "- "+f)
	}
	return nil
}

func GetInfo(cmd *cobra.Command) (*Info, error) {
	d, err := getDistro(cmd)
	if err != nil {
		return nil, err
	}

	flags := cmd.Flags()
	cache, err := flags.GetString("cache")
	if err != nil {
		return nil, err
	}
	x := &Info{
		Version: version.GetVersion(),
		Cache:   cache,
		Schemes: urlopener.Schemes,
		Distros: knownDistroNames(),
		Distro:  d.Info(),
	}
	return x, nil
}

type Info struct {
	Version string      `json:"Version"`
	Cache   string      `json:"Cache"`
	Schemes []string    `json:"Schemes"`
	Distros []string    `json:"Distros"`
	Distro  distro.Info `json:"Distro"`
}
