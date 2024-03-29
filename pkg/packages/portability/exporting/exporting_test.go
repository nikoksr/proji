package exporting

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/nikoksr/proji/pkg/api/v1/domain"
	"github.com/nikoksr/proji/pkg/packages/portability"
	"github.com/nikoksr/proji/pkg/packages/portability/importing"
	"github.com/nikoksr/proji/pkg/pointer"
)

func Test_ToConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		fileTypes []string
		dir       string
		pkg       *domain.PackageConfig
	}
	cases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid package",
			args: args{
				fileTypes: []string{portability.FileTypeJSON, portability.FileTypeTOML},
				dir:       "",
				pkg: &domain.PackageConfig{
					Label:       "tst",
					Name:        "test",
					UpstreamURL: pointer.To("https://github.com/nikoksr/proji/"),
					Description: pointer.To("A test package."),
					DirTree: &domain.DirTreeConfig{
						Entries: []*domain.DirEntryConfig{
							{IsDir: false, Path: "docs"},
							{IsDir: false, Path: "tests"},
							{IsDir: true, Path: "file1.go", Template: &domain.TemplateConfig{Path: "file1.go"}},
							{IsDir: true, Path: "file2.go", Template: &domain.TemplateConfig{Path: "file2.go"}},
							{IsDir: true, Path: "file3.go", Template: &domain.TemplateConfig{Path: "file3.go"}},
						},
					},
					Plugins: &domain.PluginSchedulerConfig{
						Pre: []*domain.PluginConfig{
							{Path: "script1.lua"},
							{Path: "script2.lua"},
						},
						Post: []*domain.PluginConfig{
							{Path: "script3.lua"},
							{Path: "script4.lua"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid destination directory",
			args: args{
				pkg: &domain.PackageConfig{},
				dir: "/invalid/dir",
			},
			wantErr: true,
		},
		{
			name: "Invalid package (nil)",
			args: args{
				pkg: nil,
				dir: "",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc

		for _, fileType := range tc.args.fileTypes {
			var configPath string
			var err error

			switch fileType {
			case portability.FileTypeTOML:
				configPath, err = ToTOML(context.Background(), tc.args.pkg, tc.args.dir)
			case portability.FileTypeJSON:
				configPath, err = ToJSON(context.Background(), tc.args.pkg, tc.args.dir)
			default:
				t.Fatalf("invalid file type: %q", fileType)
			}

			if err != nil {
				if !tc.wantErr {
					t.Fatalf("ToConfig() error = %v, wantErr %v", err, tc.wantErr)
				}
			}

			if configPath == "" {
				t.Fatalf("ToConfig() config path is empty")
			}

			t.Run(tc.name+"_"+fileType, func(t *testing.T) {
				t.Parallel()

				// Make sure we clean up after ourselves
				defer func() { _ = os.Remove(configPath) }()

				// Check if config directory is as expected.
				configDir := filepath.Dir(configPath)
				if tc.args.dir != "" {
					// If a directory was specified, it should be the same as the config directory.
					if configDir != tc.args.dir {
						t.Fatalf(
							"ToConfig() config path is not in the expected directory (-want +got):\n%s",
							cmp.Diff(tc.args.dir, filepath.Dir(configPath)),
						)
					}
				}

				// Try to load package from config.
				//
				// TODO: Is this okay? Using a different internal function for verifying the functionality of another
				//       internal function?
				pkgGot, err := importing.LocalPackage(context.Background(), configPath)
				if err != nil {
					t.Fatalf("Failed to open config created by ToConfig(): %v", err)
				}

				// Compare fields manually; didn't find a simple solution to do this with gocmp. This is good enough for now,
				// just verbose.
				if pkgGot.Label != tc.args.pkg.Label {
					t.Fatalf("ToConfig() label = %v, want %v", pkgGot.Label, tc.args.pkg.Label)
				}
				if pkgGot.Name != tc.args.pkg.Name {
					t.Fatalf("ToConfig() name = %v, want %v", pkgGot.Name, tc.args.pkg.Name)
				}

				diff := cmp.Diff(tc.args.pkg.UpstreamURL, pkgGot.UpstreamURL)
				if diff != "" {
					t.Fatalf("ToConfig() upstreamURL mismatch (-want +got):\n%s", diff)
				}

				diff = cmp.Diff(tc.args.pkg.Description, pkgGot.Description)
				if diff != "" {
					t.Fatalf("ToConfig() description mismatch (-want +got):\n%s", diff)
				}

				diff = cmp.Diff(tc.args.pkg.DirTree, pkgGot.DirTree.ToConfig())
				if diff != "" {
					t.Fatalf("ToConfig() dirTree mismatch (-want +got):\n%s", diff)
				}

				diff = cmp.Diff(tc.args.pkg.Plugins, pkgGot.Plugins.ToConfig())
				if diff != "" {
					t.Fatalf("ToConfig() plugins mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
}
