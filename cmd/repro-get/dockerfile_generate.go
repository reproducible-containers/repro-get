package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/reproducible-containers/repro-get/pkg/archutil"
	"github.com/reproducible-containers/repro-get/pkg/distro"
	"github.com/reproducible-containers/repro-get/pkg/ocidistutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func helpForBuildingDockerfiles(needsToGenerateHash bool) string {
	const tmpl = `# Copy the repro-get binary into the current directory
cp $(command -v repro-get) ./repro-get.linux-{{.OCIArchDashVariant}}

# Enable BuildKit
export DOCKER_BUILDKIT=1
{{if .NeedsToGenerateHash}}
# Generate "SHA256SUMS-{{.OCIArchDashVariant}}" in the current directory
docker build --output . -f Dockerfile.generate-hash .
{{else}}{{end}}
# Build the image
docker build .

# Clean up
rm -f ./repro-get.linux-{{.OCIArchDashVariant}}
`
	parsed, err := template.New("").Parse(tmpl)
	if err != nil {
		panic(err)
	}
	tmplArgs := map[string]interface{}{
		"OCIArchDashVariant":  archutil.OCIArchDashVariant(),
		"NeedsToGenerateHash": needsToGenerateHash,
	}
	var b bytes.Buffer
	if err = parsed.Execute(&b, tmplArgs); err != nil {
		panic(err)
	}
	return b.String()
}

func newDockerfileGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate DIR BASEIMAGE [PACKAGES]...",
		Short: `Generate Dockerfiles for "timetraveling" (EXPERIMENTAL)`,
		Long: fmt.Sprintf(`Generate Dockerfiles for "timetraveling" (EXPERIMENTAL)
- Dockerfile.generate-hash: generate the hash file "SHA256SUMS-%[1]s"
- Dockerfile:               build the image using the hash file "SHA256SUMS-%[1]s"
`, archutil.OCIArchDashVariant()),
		Example: `  # Generate "Dockerfile.generate-hash" and "Dockerfile" in the current directory for gcc
  repro-get --distro=debian dockerfile generate . debian:bullseye-20211220 gcc build-essential

  # Generate "Dockerfile" only, for consuming existing hash files
  repro-get --distro=debian dockerfile generate . debian:bullseye-20211220

To build "Dockerfile.generate-hash" and "Dockerfile":
` +
			regexp.MustCompilePOSIX("^").ReplaceAllString(helpForBuildingDockerfiles(true), "  "),
		Args: cobra.MinimumNArgs(2),
		RunE: dockerfileGenerateAction,

		DisableFlagsInUseLine: true,
	}
	return cmd
}

func dockerfileGenerateAction(cmd *cobra.Command, args []string) error {
	d, err := getDistro(cmd)
	if err != nil {
		return err
	}
	flags := cmd.Flags()
	if !flags.Changed("distro") {
		logrus.Warnf("No image distro was explicitly specified (--distro=...), assuming the distro to be %q", d.Info().Name)
	}

	providers, err := flags.GetStringSlice("provider")
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		providers = d.Info().DefaultProviders
	}

	ctx := cmd.Context()
	dir := args[0]
	baseImageOrig := args[1]
	pkgs := args[2:]

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	resolvedWithDigest, err := ocidistutil.RefWithDigest(ctx, baseImageOrig)
	if err != nil {
		return err
	}

	templateArgs := distro.DockerfileTemplateArgs{
		BaseImage:          resolvedWithDigest,
		BaseImageOrig:      baseImageOrig,
		Packages:           pkgs,
		OCIArchDashVariant: archutil.OCIArchDashVariant(),
		Providers:          providers,
	}
	opts := distro.DockerfileOpts{
		GenerateHash: len(pkgs) > 0,
	}
	if err = d.GenerateDockerfile(ctx, dir, templateArgs, opts); err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	logrus.Infof("Next steps:")
	sep := strings.Repeat("-", 5)
	fmt.Fprintln(w, sep)
	fmt.Fprint(w, helpForBuildingDockerfiles(opts.GenerateHash))
	fmt.Fprintln(w, sep)
	return nil
}
