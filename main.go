package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strings"

	"github.com/antonmedv/gitmal/pkg/git"
	"github.com/antonmedv/gitmal/pkg/templates"

	flag "github.com/spf13/pflag"
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
	flagMultipleRepo  bool
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

func processRepo(input string, outputRoot string, noFiles bool, noCommitsList bool) (templates.RepoSummary, error) {
	outputDir, err := filepath.Abs(outputRoot)
	if err != nil {
		return templates.RepoSummary{}, err
	}

	absInput, err := filepath.Abs(input)
	if err != nil {
		return templates.RepoSummary{}, err
	}
	input = absInput

	repoName := flagName
	if repoName == "" {
		repoName = strings.TrimSuffix(filepath.Base(input), ".git")
	}

	themeColor, ok := themeStyles[flagTheme]
	if !ok {
		return templates.RepoSummary{}, fmt.Errorf("invalid theme %q", flagTheme)
	}

	branchesFilter, err := regexp.Compile(flagBranches)
	if err != nil {
		return templates.RepoSummary{}, err
	}

	branches, err := git.Branches(input, branchesFilter, flagDefaultBranch)
	if err != nil {
		return templates.RepoSummary{}, err
	}

	tags, err := git.Tags(input)
	if err != nil {
		return templates.RepoSummary{}, err
	}

	defaultBranch := flagDefaultBranch
	if defaultBranch == "" {
		if containsBranch(branches, "master") {
			defaultBranch = "master"
		} else if containsBranch(branches, "main") {
			defaultBranch = "main"
		} else {
			return templates.RepoSummary{}, fmt.Errorf("No default branch found. Specify one using --default-branch flag.")
		}
	}

	if !containsBranch(branches, defaultBranch) {
		return templates.RepoSummary{}, fmt.Errorf("Default branch %q not found", defaultBranch)
	}

	if yes, a, b := hasConflictingBranchNames(branches); yes {
		return templates.RepoSummary{}, fmt.Errorf("Conflicting branchs %q and %q, both want to use %q dir name.", a, b, a.DirName())
	}

	// Start generating pages

	params := Params{
		Owner:      flagOwner,
		Name:       repoName,
		RepoDir:    input,
		OutputDir:  filepath.Join(outputDir, repoName),
		Style:      flagTheme,
		Dark:       themeColor == "dark",
		DefaultRef: git.NewRef(defaultBranch),
	}

	commits := make(map[string]git.Commit)
	commitsFor := make(map[git.Ref][]git.Commit, len(branches))
	commitsDetailsFor := make(map[git.Ref]templates.CommitDetails, len(branches))

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


		branchCommits := commitsFor[branch]
	  details := templates.CommitDetails{TotalCommits: len(branchCommits)}
		if len(branchCommits) > 0 {
			last := branchCommits[0]
			details.LastCommit = last
			details.LastCommitDate = timeAgo(last.Date)
		}
		commitsDetailsFor[branch] = details

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

	if err := generateBranches(branches, defaultBranch, params); err != nil {
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

			if branch.String() == defaultBranch {
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
	params.Ref = git.NewRef(defaultBranch)

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

	defaultRef := git.NewRef(defaultBranch)
	return buildRepoSummary(params, defaultBranch, len(branches), len(tags), commitsDetailsFor[defaultRef]), nil
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

	args := flag.Args()

	if len(args) == 0 {
		args = []string{"."}
	}

	flagMultipleRepo = len(args) > 1

	if flagPreviewThemes {
		previewThemes()
		os.Exit(0)
	}
	
	skippedRepos := []string{}
	processedRepos := []string{}
	repoSummaries := []templates.RepoSummary{}

	themeColor, themeOK := themeStyles[flagTheme]
	if !themeOK {
		echo(fmt.Sprintf("invalid theme %q", flagTheme))
		os.Exit(1)
	}
	themeDark := themeColor == "dark"

	for _, repo := range args {
		echo("Processing " + repo)

		summary, err := processRepo(repo, flagOutput, noFiles, noCommitsList)
		if err != nil {
			echo(fmt.Sprintf("Error processing %s: %v", repo, err))
			echo("Skipping " + repo)
			echo("\n ===============================\n")
			skippedRepos = append(skippedRepos, repo)
			continue
		}

		echo("Done processing " + repo)
		echo("\n ===============================\n")
		processedRepos = append(processedRepos, repo)
		repoSummaries = append(repoSummaries, summary)
	}

	if len(repoSummaries) >= 2 {
		echo("> generating portal index...")
		if err := generatePortalIndex(repoSummaries, flagOutput, flagOwner, themeDark); err != nil {
			echo(fmt.Sprintf("Error generating portal index: %v", err))
			os.Exit(1)
		}
		if flagMinify || flagGzip {
			portalIndex := filepath.Join(flagOutput, "index.html")
			echo("> post-processing portal index...")
			if err := postProcessHTMLFile(portalIndex, flagMinify, flagGzip); err != nil {
				echo(fmt.Sprintf("Error post-processing portal index: %v", err))
				os.Exit(1)
			}
		}
	}

	echo(fmt.Sprintf("Processed %d repos, skipped %d repos", len(processedRepos), len(skippedRepos)))
	if len(skippedRepos) > 0 {
		echo("Skipped repos:")
		for _, repo := range skippedRepos {
			echo(" - " + repo)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: gitmal [options] [path ...]\n")
	flag.PrintDefaults()
}
