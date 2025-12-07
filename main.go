package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"

	"github.com/antonmedv/gitmal/pkg/git"
)

var (
	flagOwner         string
	flagName          string
	flagOutput        string
	flagBranches      string
	flagDefaultBranch string
	flagTheme         string
	flagPreviewThemes bool
	flagMinify        bool
	flagGzip          bool
)

type Params struct {
	Owner      string
	Name       string
	RepoDir    string
	Ref        git.Ref
	OutputDir  string
	Style      string
	Dark       bool
	DefaultRef git.Ref
}

func main() {
	if _, ok := os.LookupEnv("GITMAL_PPROF"); ok {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		defer pprof.StopCPUProfile()
		memProf, err := os.Create("mem.prof")
		if err != nil {
			panic(err)
		}
		defer memProf.Close()
		defer pprof.WriteHeapProfile(memProf)
	}

	_, noFiles := os.LookupEnv("NO_FILES")
	_, noCommitsList := os.LookupEnv("NO_COMMITS_LIST")

	flag.StringVar(&flagOwner, "owner", "", "Project owner")
	flag.StringVar(&flagName, "name", "", "Project name")
	flag.StringVar(&flagOutput, "output", "output", "Output directory for generated HTML files")
	flag.StringVar(&flagBranches, "branches", "", "Regex for branches to include")
	flag.StringVar(&flagDefaultBranch, "default-branch", "", "Default branch to use (autodetect master or main)")
	flag.StringVar(&flagTheme, "theme", "github", "Style theme")
	flag.BoolVar(&flagPreviewThemes, "preview-themes", false, "Preview available themes")
	flag.BoolVar(&flagMinify, "minify", false, "Minify all generated HTML files")
	flag.BoolVar(&flagGzip, "gzip", false, "Compress all generated HTML files")
	flag.Usage = usage
	flag.Parse()

	input := "."
	args := flag.Args()
	if len(args) == 1 {
		input = args[0]
	}
	if len(args) > 1 {
		panic("Multiple repos not supported yet")
	}

	if flagPreviewThemes {
		previewThemes()
		os.Exit(0)
	}

	outputDir, err := filepath.Abs(flagOutput)
	if err != nil {
		panic(err)
	}

	absInput, err := filepath.Abs(input)
	if err != nil {
		panic(err)
	}
	input = absInput

	if flagName == "" {
		flagName = filepath.Base(input)
	}

	themeColor, ok := themeStyles[flagTheme]
	if !ok {
		panic("Invalid theme: " + flagTheme)
	}

	branchesFilter, err := regexp.Compile(flagBranches)
	if err != nil {
		panic(err)
	}

	branches, err := git.Branches(input, branchesFilter, flagDefaultBranch)
	if err != nil {
		panic(err)
	}

	tags, err := git.Tags(input)
	if err != nil {
		panic(err)
	}

	if flagDefaultBranch == "" {
		if containsBranch(branches, "master") {
			flagDefaultBranch = "master"
		} else if containsBranch(branches, "main") {
			flagDefaultBranch = "main"
		} else {
			echo("No default branch found. Specify one using --default-branch flag.")
			os.Exit(1)
		}
	}

	if !containsBranch(branches, flagDefaultBranch) {
		echo(fmt.Sprintf("Default branch %q not found.", flagDefaultBranch))
		echo("Specify a valid branch using --default-branch flag.")
		os.Exit(1)
	}

	// Start generating pages

	params := Params{
		Owner:      flagOwner,
		Name:       flagName,
		RepoDir:    input,
		OutputDir:  outputDir,
		Style:      flagTheme,
		Dark:       themeColor == "dark",
		DefaultRef: git.NewRef(flagDefaultBranch),
	}

	commits := make(map[string]git.Commit)
	commitsFor := make(map[git.Ref][]git.Commit, len(branches))

	for _, branch := range branches {
		commitsFor[branch], err = git.Commits(branch, params.RepoDir)
		if err != nil {
			panic(err)
		}

		for _, commit := range commitsFor[branch] {
			if alreadyExisting, ok := commits[commit.Hash]; ok && alreadyExisting.Branch == params.DefaultRef {
				continue
			}
			commit.Branch = branch
			commits[commit.Hash] = commit
		}
	}

	// Add commits from tags
	for _, tag := range tags {
		commitsForTag, err := git.Commits(git.NewRef(tag.Name), params.RepoDir)
		if err != nil {
			panic(err)
		}
		for _, commit := range commitsForTag {
			// Only add new commits
			if alreadyExisting, ok := commits[commit.Hash]; ok && !alreadyExisting.Branch.IsEmpty() {
				continue
			}
			commits[commit.Hash] = commit
		}
	}

	echo(fmt.Sprintf("> %s: %d branches, %d tags, %d commits", params.Name, len(branches), len(tags), len(commits)))

	if err := generateBranches(branches, flagDefaultBranch, params); err != nil {
		panic(err)
	}

	var defaultBranchFiles []git.Blob

	for i, branch := range branches {
		echo(fmt.Sprintf("> [%d/%d] %s@%s", i+1, len(branches), params.Name, branch))
		params.Ref = branch

		if !noFiles {
			files, err := git.Files(params.Ref, params.RepoDir)
			if err != nil {
				panic(err)
			}

			if branch.String() == flagDefaultBranch {
				defaultBranchFiles = files
			}

			err = generateBlobs(files, params)
			if err != nil {
				panic(err)
			}

			err = generateLists(files, params)
			if err != nil {
				panic(err)
			}
		}

		if !noCommitsList {
			err = generateLogForBranch(commitsFor[branch], params)
			if err != nil {
				panic(err)
			}
		}
	}

	// Back to the default branch
	params.Ref = git.NewRef(flagDefaultBranch)

	// Commits pages generation
	echo("> generating commits...")
	err = generateCommits(commits, params)
	if err != nil {
		panic(err)
	}

	// Tags page generation
	if err := generateTags(tags, params); err != nil {
		panic(err)
	}

	// Index page generation
	if !noFiles {
		if len(defaultBranchFiles) == 0 {
			panic("No files found for default branch")
		}
		err = generateIndex(defaultBranchFiles, params)
		if err != nil {
			panic(err)
		}
	}

	if flagMinify || flagGzip {
		echo("> post-processing HTML...")
		if err := postProcessHTML(params.OutputDir, flagMinify, flagGzip); err != nil {
			panic(err)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: gitmal [options] [path ...]\n")
	flag.PrintDefaults()
}
