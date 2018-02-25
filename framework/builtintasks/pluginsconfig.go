package builtintasks

import (
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/palantir/godel/framework/godellauncher"
)

func TasksConfigTask(tasksCfgInfo godellauncher.TasksConfigInfo) godellauncher.Task {
	return godellauncher.CobraCLITask(&cobra.Command{
		Use:   "tasks-config",
		Short: "Prints the full YAML configuration used to load tasks and assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printTasksCfgInfo(tasksCfgInfo, cmd.OutOrStdout())
		},
	})
}

func printTasksCfgInfo(tasksCfgInfo godellauncher.TasksConfigInfo, stdout io.Writer) error {
	if err := printWithHeader("Built-in plugin configuration", tasksCfgInfo.BuiltinPluginsConfig, stdout); err != nil {
		return err
	}
	fmt.Fprintln(stdout)

	if err := printWithHeader("Fully resolved godel tasks configuration", tasksCfgInfo.TasksConfig, stdout); err != nil {
		return err
	}
	fmt.Fprintln(stdout)

	if err := printWithHeader("Plugin configuration for default tasks", tasksCfgInfo.DefaultTasksPluginsConfig, stdout); err != nil {
		return err
	}
	return nil
}

func printWithHeader(header string, in interface{}, stdout io.Writer) error {
	printHeader(header, stdout)
	ymlString, err := toYAMLString(in)
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, ymlString)
	return nil
}

func printHeader(header string, stdout io.Writer) {
	fmt.Fprintln(stdout, header+":")
	fmt.Fprintln(stdout, strings.Repeat("-", len(header)+1))
}

func toYAMLString(in interface{}) (string, error) {
	ymlBytes, err := yaml.Marshal(in)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal YAML")
	}
	return string(ymlBytes), nil
}