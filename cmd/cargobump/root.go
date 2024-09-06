package cargobump

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"chainguard.dev/apko/pkg/log"
	charmlog "github.com/charmbracelet/log"
	"github.com/hectorj2f/cargobump/pkg"
	"github.com/hectorj2f/cargobump/pkg/parser"
	"github.com/hectorj2f/cargobump/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/release-utils/version"
)

type rootCLIFlags struct {
	packages  string
	bumpFile  string
	cargoRoot string
}

var rootFlags rootCLIFlags

func New() *cobra.Command {
	var logPolicy []string
	var level log.CharmLogLevel

	cmd := &cobra.Command{
		Use:   "cargobump <file-to-bump>",
		Short: "cargobump cli",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			out, err := log.Writer(logPolicy)
			if err != nil {
				return fmt.Errorf("failed to create log writer: %w", err)
			}
			slog.SetDefault(slog.New(charmlog.NewWithOptions(out, charmlog.Options{ReportTimestamp: true, Level: charmlog.Level(level)})))

			return nil
		},

		// Uncomment the following line if your bare application
		// has an action associated with it:
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootFlags.packages == "" && rootFlags.bumpFile == "" {
				return fmt.Errorf("no packages or bump file provides, use --packages/--bump-file")
			}

			if rootFlags.bumpFile != "" && rootFlags.packages != "" {
				return fmt.Errorf("use either --packages or --bump-file")
			}
			var file *os.File
			parse := parser.NewParser()
			var patches map[string]*types.Package
			if rootFlags.bumpFile != "" {
				var err error
				file, err = os.Open(rootFlags.bumpFile)
				if err != nil {
					return fmt.Errorf("failed reading file: %w", err)
				}
				defer file.Close()
				patches, err = parse.ParseBumpFile(file)
				if err != nil {
					return fmt.Errorf("failed to parse the bump file: %w", err)
				}
			} else {
				ps := strings.Split(rootFlags.packages, " ")
				for _, pkg := range ps {
					parts := strings.Split(pkg, "@")
					if len(parts) != 2 {
						return fmt.Errorf("Error: Invalid package format. Each package should be in the format <package@version>. Usage: cargobump --packages=\"<package1@version> <package2@version> ...\"")
					}
					patches[parts[0]] = &types.Package{
						Name:    parts[0],
						Version: parts[1],
					}
				}
			}

			cargoLockFile, err := os.Open(filepath.Join(rootFlags.cargoRoot, "Cargo.lock"))
			if err != nil {
				return fmt.Errorf("failed reading file: %w", err)
			}
			defer cargoLockFile.Close()

			pkgs, err := parse.ParseCargoLock(cargoLockFile)
			if err != nil {
				return fmt.Errorf("failed to parse Cargo.lock file: %w", err)
			}

			if err = pkg.Update(patches, pkgs, rootFlags.cargoRoot); err != nil {
				return fmt.Errorf("failed to parse the pom file: %w", err)
			}

			return nil
		},
	}
	cmd.PersistentFlags().StringSliceVar(&logPolicy, "log-policy", []string{"builtin:stderr"}, "log policy (e.g. builtin:stderr, /tmp/log/foo)")
	cmd.PersistentFlags().Var(&level, "log-level", "log level (e.g. debug, info, warn, error)")

	cmd.AddCommand(version.WithFont("starwars"))

	cmd.DisableAutoGenTag = true

	flagSet := cmd.Flags()
	flagSet.StringVar(&rootFlags.cargoRoot, "cargoroot", "", "path to the Cargo.lock root")
	flagSet.StringVar(&rootFlags.packages, "packages", "", "A space-separated list of dependencies to update in form package@version")
	flagSet.StringVar(&rootFlags.bumpFile, "bump-file", "", "The input file to read dependencies to bump from")
	return cmd
}
