// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package command provides handling to generate new scaffolding, compile, and
// publish CLI plugins.
package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"

	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	rtplugin "github.com/vmware-tanzu/tanzu-plugin-runtime/plugin"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/types"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func init() {
	// Show timestamp as part of the logs
	log.ShowTimestamp(true)
	// Redirect the logs to stdout for the user to consume
	log.SetStderr(os.Stdout)
}

var (
	version, artifactsDir, ldflags string
	tags, goprivate                string
	targetArch                     []string
	groupByOSArch                  bool
)

type plugin struct {
	rtplugin.PluginDescriptor
	path     string
	testPath string
	docPath  string
	modPath  string
	buildID  string
	target   string
}

// PluginCompileArgs contains the values to use for compiling plugins.
type PluginCompileArgs struct {
	Version                    string
	SourcePath                 string
	ArtifactsDir               string
	LDFlags                    string
	Tags                       string
	Match                      string
	Description                string
	GoPrivate                  string
	PluginScopeAssociationFile string
	TargetArch                 []string
	GroupByOSArch              bool
}

const local = "local"

var minConcurrent = 2
var identifiers = []string{
	string('\U0001F435'),
	string('\U0001F43C'),
	string('\U0001F436'),
	string('\U0001F430'),
	string('\U0001F98A'),
	string('\U0001F431'),
	string('\U0001F981'),
	string('\U0001F42F'),
	string('\U0001F42E'),
	string('\U0001F437'),
	string('\U0001F42D'),
	string('\U0001F428'),
}

func getID(i int) string {
	index := i
	if i >= len(identifiers) {
		// Well aren't you lucky
		index = i % len(identifiers)
	}
	return identifiers[index]
}

func getBuildArch(arch []string) []cli.Arch {
	var arrArch []cli.Arch
	for _, buildArch := range arch {
		if buildArch == string(AllTargets) {
			for arch := range archMap {
				arrArch = append(arrArch, arch)
			}
		} else if buildArch == local {
			arrArch = append(arrArch, cli.BuildArch())
		} else {
			arrArch = append(arrArch, cli.Arch(buildArch))
		}
	}
	return arrArch
}

func getMaxParallelism() int {
	maxConcurrent := runtime.NumCPU() - 2
	if maxConcurrent < minConcurrent {
		maxConcurrent = minConcurrent
	}
	return maxConcurrent
}

type errInfo struct {
	Err  error
	Path string
	ID   string
}

// setGlobals initializes a set of global variables used throughout the compile
// process, based on the arguments passed in.
func setGlobals(compileArgs *PluginCompileArgs) {
	version = compileArgs.Version
	artifactsDir = compileArgs.ArtifactsDir
	ldflags = compileArgs.LDFlags
	tags = compileArgs.Tags
	goprivate = compileArgs.GoPrivate
	targetArch = compileArgs.TargetArch
	groupByOSArch = compileArgs.GroupByOSArch

	// Append version specific ldflag by default so that user doesn't need to pass this ldflag always.
	ldflags = fmt.Sprintf("%s -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Version=%s'", ldflags, version)
}

func Compile(compileArgs *PluginCompileArgs) error {
	// Set our global values based on the passed args
	setGlobals(compileArgs)

	log.Infof("building local repository at %s, %v, %v", compileArgs.ArtifactsDir, compileArgs.Version, compileArgs.TargetArch)

	manifest := cli.Manifest{
		CreatedTime: time.Now(),
		Plugins:     []cli.Plugin{},
	}

	files, err := os.ReadDir(compileArgs.SourcePath)
	if err != nil {
		return err
	}

	// Limit the number of concurrent operations we perform so we don't overwhelm the system.
	maxConcurrent := getMaxParallelism()
	guard := make(chan struct{}, maxConcurrent)

	// Mix up IDs so we don't always get the same set.
	randSkew := rand.Intn(len(identifiers)) // nolint:gosec
	var wg sync.WaitGroup
	plugins := make(chan cli.Plugin, len(files))
	fatalErrors := make(chan errInfo, len(files))
	g := glob.MustCompile(compileArgs.Match)
	for i, f := range files {
		if f.IsDir() {
			if g.Match(f.Name()) {
				wg.Add(1)
				guard <- struct{}{}
				go func(fullPath, id string) {
					defer wg.Done()
					p, err := buildPlugin(fullPath, id)
					if err != nil {
						fatalErrors <- errInfo{Err: err, Path: fullPath, ID: id}
					} else {
						plug := cli.Plugin{
							Name:        p.Name,
							Description: p.Description,
							Target:      p.target,
							Versions:    []string{p.Version},
						}
						plugins <- plug
					}
					<-guard
				}(filepath.Join(compileArgs.SourcePath, f.Name()), getID(i+randSkew))
			}
		}
	}

	wg.Wait()
	close(plugins)
	close(fatalErrors)
	log.Info("========")

	hasFailed := false

	var exerr *exec.ExitError
	for err := range fatalErrors {
		hasFailed = true

		if errors.As(err.Err, &exerr) {
			log.Errorf("%s - building plugin %q failed - %v:\n%s", err.ID, err.Path, err.Err, exerr.Stderr)
		} else {
			log.Errorf("%s - building plugin %q failed - %v", err.ID, err.Path, err.Err)
		}
	}

	if hasFailed {
		os.Exit(1)
	}

	for plug := range plugins {
		manifest.Plugins = append(manifest.Plugins, plug)
	}

	err = savePluginManifest(manifest, compileArgs.ArtifactsDir, compileArgs.GroupByOSArch)
	if err != nil {
		return err
	}

	err = savePluginGroupManifest(manifest.Plugins, compileArgs.ArtifactsDir, compileArgs.PluginScopeAssociationFile, compileArgs.GroupByOSArch)
	if err != nil {
		return err
	}

	log.Success("successfully built local repository")
	return nil
}

func buildPlugin(path, id string) (plugin, error) {
	log.Infof("%s - building plugin at path %q", id, path)

	var modPath string

	cmd := goCommand("run", "-ldflags", ldflags, "-tags", tags)

	if isLocalGoModFileExists(path) {
		modPath = path
		cmd.Dir = modPath
		cmd.Args = append(cmd.Args, "./.")
		log.Infof("%s - running godep path %q", id, path)
		err := runDownloadGoDep(path, id)
		if err != nil {
			log.Errorf("%s - cannot download go dependencies in path: %s - error: %v", id, path, err)
			return plugin{}, err
		}
	} else {
		modPath = ""
		cmd.Args = append(cmd.Args, fmt.Sprintf("./%s", path))
	}

	cmd.Args = append(cmd.Args, "info")
	b, err := cmd.Output()

	if err != nil {
		log.Errorf("%s - error: %v", id, err)
		log.Errorf("%s - output: (%v)", id, string(b))
		return plugin{}, err
	}

	var desc rtplugin.PluginDescriptor
	err = json.Unmarshal(b, &desc)
	if err != nil {
		log.Errorf("%s - error unmarshalling plugin descriptor: %v", id, err)
		return plugin{}, err
	}

	testPath := filepath.Join(path, "test")
	_, err = os.Stat(testPath)
	if err != nil {
		if os.Getenv("TZ_ENFORCE_TEST_PLUGIN") == "1" {
			log.Errorf("%s - plugin %q must implement test", id, desc.Name)
			return plugin{}, err
		}
		testPath = ""
	}

	docPath := filepath.Join(path, "README.md")
	_, err = os.Stat(docPath)
	if err != nil {
		log.Errorf("%s - plugin %q requires a README.md file", id, desc.Name)
		return plugin{}, err
	}

	target, err := getPluginTarget(&desc, path, id)
	if err != nil {
		return plugin{}, err
	}

	p := plugin{
		PluginDescriptor: desc,
		docPath:          docPath,
		buildID:          id,
		target:           target,
	}

	if modPath != "" {
		p.path = "."
		if testPath == "" {
			p.testPath = ""
		} else {
			p.testPath = "test"
		}
		p.modPath = modPath
	} else {
		p.path = path
		p.testPath = testPath
		p.modPath = ""
	}

	log.V(4).Infof("plugin %v", p)

	err = p.compile()
	if err != nil {
		log.Errorf("%s - error compiling plugin %s", id, desc.Name)
		return plugin{}, err
	}

	return p, nil
}

type target struct {
	env  []string
	args []string
}

func (t target) build(targetPath, prefix, modPath, ldflags, tags string) error {
	cmd := goCommand("build")

	var commonArgs = []string{
		"-ldflags", ldflags,
		"-tags", tags,
	}

	cmd.Args = append(cmd.Args, t.args...)
	cmd.Args = append(cmd.Args, commonArgs...)

	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, t.env...)

	if modPath != "" {
		cmd.Dir = modPath
	}

	cmd.Args = append(cmd.Args, fmt.Sprintf("./%s", targetPath))

	log.Infof("%s$ %s", prefix, cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("%serror: %v", prefix, err)
		log.Errorf("%soutput: %v", prefix, string(output))
		return err
	}
	return nil
}

// AllTargets are all the known targets.
const AllTargets cli.Arch = "all"

type targetBuilder func(pluginName, outPath string) target

var archMap = map[cli.Arch]targetBuilder{
	cli.Linux386: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"CGO_ENABLED=0",
				"GOARCH=386",
				"GOOS=linux",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.Linux386)),
			},
		}
	},
	cli.LinuxAMD64: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"CGO_ENABLED=0",
				"GOARCH=amd64",
				"GOOS=linux",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.LinuxAMD64)),
			},
		}
	},
	cli.LinuxARM64: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"CGO_ENABLED=0",
				"GOARCH=arm64",
				"GOOS=linux",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.LinuxARM64)),
			},
		}
	},
	cli.DarwinAMD64: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"GOARCH=amd64",
				"GOOS=darwin",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.DarwinAMD64)),
			},
		}
	},
	cli.DarwinARM64: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"GOARCH=arm64",
				"GOOS=darwin",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.DarwinARM64)),
			},
		}
	},
	cli.Win386: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"GOARCH=386",
				"GOOS=windows",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.Win386)),
			},
		}
	},
	cli.WinAMD64: func(pluginName, outPath string) target {
		return target{
			env: []string{
				"GOARCH=amd64",
				"GOOS=windows",
			},
			args: []string{
				"-o", filepath.Join(outPath, cli.MakeArtifactName(pluginName, cli.WinAMD64)),
			},
		}
	},
}

func (p *plugin) compile() error {
	absArtifactsDir, err := filepath.Abs(artifactsDir)
	if err != nil {
		return err
	}

	err = buildTargets(p.path, absArtifactsDir, p.Name, p.target, p.buildID, p.modPath, false)
	if err != nil {
		return err
	}

	if p.testPath != "" {
		err = buildTargets(p.testPath, absArtifactsDir, p.Name, p.target, p.buildID, p.modPath, true)
		if err != nil {
			return err
		}
	}

	if !groupByOSArch {
		b, err := yaml.Marshal(p.PluginDescriptor)
		if err != nil {
			return err
		}

		configPath := filepath.Join(absArtifactsDir, p.Name, cli.PluginDescriptorFileName)
		err = os.WriteFile(configPath, b, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildTargets(targetPath, artifactsDir, pluginName, target, id, modPath string, isTest bool) error {
	if id != "" {
		id = fmt.Sprintf("%s - ", id)
	}

	targets := map[cli.Arch]targetBuilder{}
	for _, buildArch := range targetArch {
		if buildArch == string(AllTargets) {
			targets = archMap
		} else if buildArch == local {
			localArch := cli.BuildArch()
			targets[localArch] = archMap[localArch]
		} else {
			bArch := cli.Arch(buildArch)
			if val, ok := archMap[bArch]; !ok {
				log.Errorf("%q build architecture is not supported", buildArch)
			} else {
				targets[cli.Arch(buildArch)] = val
			}
		}
	}

	for arch, targetBuilder := range targets {
		pn := pluginName

		outputDir := artifactsDir
		if groupByOSArch {
			outputDir = filepath.Join(outputDir, arch.OS(), arch.Arch(), target)
		}
		outputDir = filepath.Join(outputDir, pn, version)
		if isTest {
			outputDir = filepath.Join(outputDir, "test")
			pn = fmt.Sprintf("%s-test", pn)
		}

		tgt := targetBuilder(pn, outputDir)
		err := tgt.build(targetPath, id, modPath, ldflags, tags)
		if err != nil {
			return err
		}
	}
	return nil
}

func runDownloadGoDep(targetPath, prefix string) error {
	cmdgomoddownload := goCommand("mod", "download")
	cmdgomoddownload.Dir = targetPath

	log.Infof("%s$ %s", prefix, cmdgomoddownload.String())
	output, err := cmdgomoddownload.CombinedOutput()
	if err != nil {
		log.Errorf("%serror: %v", prefix, err)
		log.Errorf("%soutput: %v", prefix, string(output))
		return err
	}
	return nil
}

func isLocalGoModFileExists(path string) bool {
	_, err := os.Stat(filepath.Join(path, "go.mod"))
	return err == nil
}

func goCommand(arg ...string) *exec.Cmd {
	cmd := exec.Command("go", arg...)
	if goprivate != "" {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOPRIVATE=%s", goprivate))
	}
	return cmd
}

func getPluginTarget(pd *rtplugin.PluginDescriptor, pluginPath, id string) (string, error) {
	if !groupByOSArch || pd == nil {
		return "", nil
	}

	// If target is specified in the plugin descriptor use it
	if configtypes.IsValidTarget(string(pd.Target), true, false) {
		return string(pd.Target), nil
	}

	// If target is not specified in the plugin descriptor check `metadata.yaml` in the
	// plugin directory to get the target of the plugin
	metadataFilePath := filepath.Join(pluginPath, "metadata.yaml")
	b, err := os.ReadFile(metadataFilePath)
	if err != nil {
		log.Errorf("%s - plugin %q requires a metadata.yaml file", id, pd.Name)
		return "", err
	}
	var metadata types.Metadata
	err = yaml.Unmarshal(b, &metadata)
	if err != nil {
		log.Errorf("%s - error unmarshalling plugin metadata.yaml file: %v", id, err)
		return "", err
	}
	return metadata.Target, nil
}

func savePluginManifest(manifest cli.Manifest, artifactsDir string, groupByOSArch bool) error {
	b, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}

	if groupByOSArch {
		arrTargetArch := getBuildArch(targetArch)
		arrTargetArch = append(arrTargetArch, "")
		for _, osarch := range arrTargetArch {
			manifestPath := filepath.Join(artifactsDir, osarch.OS(), osarch.Arch(), cli.PluginManifestFileName)
			err := os.WriteFile(manifestPath, b, 0644)
			if err != nil {
				return err
			}
		}
	} else {
		manifestPath := filepath.Join(artifactsDir, cli.ManifestFileName)
		err = os.WriteFile(manifestPath, b, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func savePluginGroupManifest(plugins []cli.Plugin, artifactsDir, pluginScopeAssociationFile string, groupByOSArch bool) error {
	if !groupByOSArch || pluginScopeAssociationFile == "" {
		return nil
	}

	log.Info("saving plugin group manifest...")

	b, err := os.ReadFile(pluginScopeAssociationFile)
	if err != nil {
		return err
	}

	psm := &cli.PluginScopeMetadata{}
	err = yaml.Unmarshal(b, psm)
	if err != nil {
		return err
	}

	getPluginScope := func(name, target string) bool {
		for _, p := range psm.Plugins {
			if p.Name == name && p.Target == target {
				return p.IsContextScoped
			}
		}
		return false
	}

	pgManifest := cli.PluginGroupManifest{
		CreatedTime: time.Now(),
		Plugins:     []cli.PluginNameTargetScopeVersion{},
	}

	for _, plug := range plugins {
		pluginNTSV := cli.PluginNameTargetScopeVersion{}
		pluginNTSV.Name = plug.Name
		pluginNTSV.Target = plug.Target
		pluginNTSV.IsContextScoped = getPluginScope(plug.Name, plug.Target)
		pluginNTSV.Version = plug.Versions[0]

		pgManifest.Plugins = append(pgManifest.Plugins, pluginNTSV)
	}

	b, err = yaml.Marshal(pgManifest)
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(artifactsDir, cli.PluginGroupManifestFileName)
	err = os.WriteFile(manifestPath, b, 0644)
	if err != nil {
		return err
	}

	return nil
}
