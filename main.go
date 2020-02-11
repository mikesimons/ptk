package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/flosch/pongo2"
	"github.com/imdario/mergo"

	"github.com/ansel1/merry"
	"github.com/spf13/cobra"
)

// ptk path parents . | ptk lf append terraform.tfvars.json | ptk lf file --exists

// ptk download https://testfile --checksum sha256: --output <file|->
// Also s3://

// ptk lines replace --in-place "regexp-pattern" "replacement" [file]
// ptk lines insert --in-place --before=<pattern> --after=<pattern> <line> [file]
// ptk lines remove --in-place line [file]

// ptk template --var-file <file> sometemplate.tpl --output <file|->

// ptk package -v '1.14' nginx
// ptk apt-repo <url>

// ptk service name --command ...

var versionString = "dev"

func main() {
	cmd := &cobra.Command{
		Use:     "ptk",
		Version: versionString,
		PreRun: func(cmd *cobra.Command, args []string) {
			merry.SetStackCaptureEnabled(true)
		},
	}

	cmd.AddCommand(pathCommand())
	cmd.AddCommand(filterCommand())
	cmd.AddCommand(templateCommand())

	cmd.Execute()
}

func pathCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "path",
	}

	cmd.AddCommand(pathParentsCommand())

	return cmd
}

func pathParentsCommand() *cobra.Command {
	return &cobra.Command{
		Use: "parents",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				pwd, err := os.Getwd()
				if err != nil {
					log.Fatal("Could not get PWD")
				}

				args = append(args, pwd)
			}

			path := args[0]

			oldPath := ""
			for oldPath != path {
				fmt.Println(path)
				oldPath = path
				path = filepath.Dir(path)

			}
		},
	}
}

func filterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lines",
		Aliases: []string{"l"},
	}

	cmd.AddCommand(filterFileCommand())
	cmd.AddCommand(filterAppendCommand())
	cmd.AddCommand(filterReplaceCommand())

	return cmd
}

func filterFileCommand() *cobra.Command {
	// exists
	// readable
	// file
	// directory
	// owner
	// group
	//
	var existsFilter bool
	var notExistsFilter bool

	cmd := &cobra.Command{
		Use: "file",
		Run: func(cmd *cobra.Command, args []string) {
			bytes, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Could not read input")
			}

			data := strings.Split(string(bytes), "\n")
			for _, line := range data {
				_, err := os.Stat(line)

				if existsFilter {
					if os.IsNotExist(err) {
						continue
					}
				}

				if notExistsFilter {
					if !os.IsNotExist(err) {
						continue
					}
				}

				fmt.Println(line)
			}
		},
	}

	cmd.Flags().BoolVar(&existsFilter, "exists", false, "Print files that exist")
	cmd.Flags().BoolVar(&notExistsFilter, "not-exists", false, "Print files that don't exist")

	return cmd
}

func filterAppendCommand() *cobra.Command {
	var asPath bool
	cmd := &cobra.Command{
		Use: "append",
		Run: func(cmd *cobra.Command, args []string) {
			bytes, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Could not read input")
			}

			data := strings.Split(string(bytes), "\n")
			for _, line := range data {
				if asPath {
					fmt.Println(filepath.Join(line, args[0]))
				} else {
					fmt.Printf("%s%s\n", line, args[0])
				}
			}
		},
	}

	cmd.Flags().BoolVar(&asPath, "as-path", false, "Manage strings as paths")

	return cmd
}

func filterReplaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "replace",
		Run: func(cmd *cobra.Command, args []string) {
			bytes, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Could not read input")
			}

			pattern := regexp.MustCompile(args[0])
			data := strings.Split(string(bytes), "\n")
			for _, line := range data {
				line = pattern.ReplaceAllString(line, args[1])
				fmt.Println(line)
			}
		},
	}

	return cmd
}

func templateCommand() *cobra.Command {
	var data []string
	cmd := &cobra.Command{
		Use:     "template",
		Aliases: []string{"tpl"},
		Run: func(cmd *cobra.Command, args []string) {
			pongo2.RegisterFilter("base64encode", filterBase64Encode)

			if len(args) != 1 || len(data) < 1 {
				log.Fatalf("Usage: ptk tpl <template> --data='file://...' --data='json://{}' --data='yaml://{} ...")
			}

			context := make(pongo2.Context)

			file, err := os.Open(args[0])
			defer file.Close()
			if err != nil {
				log.Fatalf("Could not open template file")
			}

			template, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("Could not read template")
			}

			for _, d := range data {
				tmp := make(map[string]interface{})
				if strings.HasPrefix(d, "file://") {
					f, err := os.Open(strings.TrimPrefix(d, "file://"))
					defer f.Close()
					if err != nil {
						log.Fatalf("Could not open data file")
					}

					bytes, err := ioutil.ReadAll(f)
					if err != nil {
						log.Fatalf("Could not read data file")
					}

					err = yaml.Unmarshal(bytes, &tmp)
					if err != nil {
						log.Fatalf("Could not parse data file")
					}
				}

				if strings.HasPrefix(d, "yaml://") || strings.HasPrefix(d, "json://") {
					str := strings.TrimPrefix(strings.TrimPrefix(d, "yaml://"), "json://")
					err := yaml.Unmarshal([]byte(str), &tmp)
					if err != nil {
						log.Fatalf("Could not parse data arg")
					}
				}

				mergo.Map(&context, tmp)
			}

			tpl, err := pongo2.FromString(string(template))
			if err != nil {
				log.Fatalf("Could not parse template")
			}

			out, err := tpl.Execute(context)
			if err != nil {
				log.Fatalf("Could not render template")
			}

			fmt.Print(out)
		},
	}
	cmd.Flags().StringArrayVar(&data, "data", data, "data")
	return cmd
}

func filterBase64Encode(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return pongo2.AsValue(base64.StdEncoding.EncodeToString([]byte(in.String()))), nil
}

func downloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "download",
		Run: func(cmd *cobra.Command, args []string) {
			// hashicorp getter fetch file to temp
			// if set, hashicorp getter fetch checksum file; extract checksum for downloaded filename
			// if set, hashicorp getter fetch signature file
			// Check checksum (if set); fail if invalid
			// Check signature (if set); fail if invalid
			// Copy to target
		},
	}

	return cmd
}
