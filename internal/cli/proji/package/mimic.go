package pkg

import (
	"context"
	"regexp"

	"github.com/cockroachdb/errors"
	"github.com/nikoksr/simplog"
	"github.com/spf13/cobra"

	"github.com/nikoksr/proji/internal/cli"
)

func newMimicCommand() *cobra.Command {
	var exclude string

	cmd := &cobra.Command{
		Use:                   "mimic [OPTIONS] PATH [PATH...]",
		Short:                 "Create packages that mimic local directories or remote repositories",
		Args:                  cobra.MinimumNArgs(1),
		DisableFlagsInUseLine: true,

		Example: `  mimic https://github.com/nikoksr/my_repo
  mimic https://github.com/nikoksr/my_repo/tree/my_branch
  mimic gh:nikoksr/my_repo@my_branch
  mimic ./some_dir`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return mimicPackages(cmd.Context(), exclude, args...)
		},
	}

	cmd.Flags().StringVarP(&exclude, "exclude", "e", "", "Regex pattern to exclude paths")

	return cmd
}

func mimicPackages(ctx context.Context, exclude string, paths ...string) error {
	logger := simplog.FromContext(ctx)

	// Get package manager from session
	logger.Debug("getting package manager from cli session")
	session := cli.SessionFromContext(ctx)
	pama := session.PackageManager
	if pama == nil {
		return errors.New("no package manager available")
	}

	// Compile regex pattern for excluding paths. Value from flag has priority over value from config.
	if exclude == "" {
		exclude = session.Config.Import.Exclude
	}

	reExclude, err := regexp.Compile(exclude)
	if err != nil {
		return errors.Wrap(err, "compile exclude regexp")
	}

	// Mimicking packages
	logger.Debugf("mimicking %d packages", len(paths))
	for _, path := range paths {
		if path == "" {
			logger.Debug("skipping empty path")
			continue
		}

		logger.Debugf("mimicking package %q", path)
		pkg, err := importPackage(ctx, path, false, reExclude)
		if err != nil {
			return errors.Wrapf(err, "import package as mimic from %q", path)
		}

		logger.Debugf("adding package %q", pkg.Label)
		if err = pama.Store(ctx, pkg); err != nil {
			return errors.Wrapf(err, "store %q, imported as mimic of %q", pkg.Name, path)
		}

		logger.Infof("Successfully installed package %q as %q", pkg.Name, pkg.Label)
	}

	return nil
}
