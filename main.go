package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

// ANSI escape code for colors
const (
	ColorGreen   = "\033[32m"
	ColorRed     = "\033[31m"
	ColorYellow  = "\033[33m"
	ColorReset   = "\033[0m"
)

// Regex to remove ANSI color codes
var ansiStripper = regexp.MustCompile("\033[[0-9;]*m")

type BranchDetail struct {
	Name    string
	Hash    string
	Author  string
	Date    string
	Message string
}

// cleanBranchName removes color codes and merge indicators from a branch name
func cleanBranchName(branchName string) string {
	// First, remove ANSI color codes
	cleaned := ansiStripper.ReplaceAllString(branchName, "")
	// Then, remove the merge indicator (e.g., " (merged)" or " (unmerged)")
	parts := strings.SplitN(cleaned, " (", 2)
	return strings.TrimSpace(parts[0])
}

func getRemoteBranchDetail(branchName string) (BranchDetail, error) {
	cleanName := cleanBranchName(branchName)
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%H%n%an%n%ad%n%s", cleanName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BranchDetail{}, fmt.Errorf("git log failed: %w\n%s", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 4 {
		return BranchDetail{}, fmt.Errorf("unexpected git log output: %s", string(output))
	}

	return BranchDetail{
		Name:    cleanName,
		Hash:    lines[0],
		Author:  lines[1],
		Date:    lines[2],
		Message: lines[3],
	}, nil
}

func isMergedToHead(branch string) bool {
	cmd := exec.Command("git", "branch", "-r", "--merged", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log error but continue, as this is not critical
		fmt.Fprintf(os.Stderr, "Warning: Could not get merged branches: %v\n", err)
		return false
	}
	mergedBranches := strings.Split(string(output), "\n")
	for _, mergedBranch := range mergedBranches {
		if strings.TrimSpace(mergedBranch) == strings.TrimSpace(branch) {
			return true
		}
	}
	return false
}

// isProtectedBranch checks if a given branch name is a protected branch (e.g., main, master)
func isProtectedBranch(branchName string) bool {
	protectedBranches := []string{"main", "master"}

	// Extract just the branch name without the remote prefix (e.g., "origin/main" -> "main")
	parts := strings.SplitN(branchName, "/", 2)
	cleanBranchName := branchName
	if len(parts) == 2 {
		cleanBranchName = parts[1]
	}

	for _, protected := range protectedBranches {
		if cleanBranchName == protected {
			return true
		}
	}
	return false
}

func main() {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	bundle.LoadMessageFileFS(localeFS, "locales/en.json")
	bundle.LoadMessageFileFS(localeFS, "locales/ja.json")

	langFlag := flag.String("lang", "", "Specify the language (e.g., en, ja)")
	helpFlag := flag.Bool("h", false, "Show help")
	flag.BoolVar(helpFlag, "help", false, "Show help")

	// Internal flag for fzf preview
	getLogFlag := flag.String("get-remote-log", "", "Internal flag to get log for a remote branch")

	flag.Parse()

	var selectedLang string
	if *langFlag != "" {
		selectedLang = *langFlag
	} else {
		envLang := os.Getenv("LANG")
		if strings.Contains(envLang, "ja") {
			selectedLang = "ja"
		} else {
			selectedLang = "en" // Default to English if parsing fails
		}
	}


	// Fallback to English if the selected language is not explicitly supported
	if selectedLang != "ja" {
		selectedLang = "en"
	}

	localizer := i18n.NewLocalizer(bundle, selectedLang)

	// Handle internal fzf preview request
	if *getLogFlag != "" {
		cleanName := cleanBranchName(*getLogFlag)
		cmd := exec.Command("git", "log", "--color=always", cleanName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting log for %s: %v\n", cleanName, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *helpFlag {
		usage, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "HelpUsage"})
		description, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "HelpDescription"})
		help, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "HelpFlag"})
		langHelp, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "HelpLangFlag"})

		fmt.Printf("%s\n\n%s\n\nOptions:\n  -h, --help    %s\n  -lang string  %s\n", usage, description, help, langHelp)
		os.Exit(0)
	}

	// Check if fzf is installed
	if _, err := exec.LookPath("fzf"); err != nil {
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "FzfNotFound"}))
		fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "InstallFzf"}))
		os.Exit(1)
	}

	// Get all remote branches
	cmd := exec.Command("git", "branch", "-r")
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg, _ := localizer.Localize(&i18n.LocalizeConfig{
			MessageID: "ErrorGettingRemoteBranches",
			TemplateData: map[string]interface{}{"Error": err},
		})
		fmt.Println(msg)
		os.Exit(1)
	}

	allRemoteBranches := strings.Split(string(output), "\n")

	var fzfItems []string
	for _, branch := range allRemoteBranches {
		trimmedBranch := strings.TrimSpace(branch)
		if trimmedBranch != "" && !strings.HasSuffix(trimmedBranch, "/HEAD") {
			var indicator string
			var color string

			if isProtectedBranch(trimmedBranch) {
				indicator = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "ProtectedIndicator"})
				color = ColorYellow
			} else if isMergedToHead(trimmedBranch) {
				indicator = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "MergedIndicator"})
				color = ColorGreen
			} else {
				indicator = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "UnmergedIndicator"})
				color = ColorRed
			}
			fzfItems = append(fzfItems, fmt.Sprintf("%s%s %s%s", color, trimmedBranch, indicator, ColorReset))
		}
	}

	if len(fzfItems) == 0 {
		msg, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "NoRemoteBranches"})
		fmt.Println(msg)
		os.Exit(0)
	}

	// Prepare fzf command
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	fzfCmd := exec.Command("fzf", "--multi", "--ansi", "--preview", fmt.Sprintf("%s -get-remote-log {}", executablePath))
	fzfCmd.Stderr = os.Stderr // Show fzf errors

	// Pass branches to fzf stdin
	fzfStdin, err := fzfCmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdin pipe for fzf: %v\n", err)
		os.Exit(1)
	}
	go func() {
		defer fzfStdin.Close()
		for _, item := range fzfItems {
			fmt.Fprintln(fzfStdin, item)
		}
	}()

	// Capture fzf stdout
	var fzfStdout bytes.Buffer
	fzfCmd.Stdout = &fzfStdout

	// Run fzf
	err = fzfCmd.Run()
	if err != nil {
		// fzf returns non-zero exit code if no selection or cancelled
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 130 {
			// User cancelled (Ctrl+C or Esc)
			fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "DeletionCancelled"}))
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error running fzf: %v\n", err)
		os.Exit(1)
	}

	selectedBranchesStr := strings.TrimSpace(fzfStdout.String())
	if selectedBranchesStr == "" {
		msg, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "NoBranchesSelected"})
		fmt.Println(msg)
		os.Exit(0)
	}

	// Clean selected branch names and filter out protected branches
	var branchesToDelete []string
	var protectedBranchesSelected []string
	for _, selectedItem := range strings.Split(selectedBranchesStr, "\n") {
		cleanedBranch := cleanBranchName(selectedItem)
		if isProtectedBranch(cleanedBranch) {
			protectedBranchesSelected = append(protectedBranchesSelected, cleanedBranch)
		} else {
			branchesToDelete = append(branchesToDelete, cleanedBranch)
		}
	}

	// Notify user about skipped protected branches
	for _, protectedBranch := range protectedBranchesSelected {
		msg, _ := localizer.Localize(&i18n.LocalizeConfig{
			MessageID: "ProtectedBranchSkipped",
			TemplateData: map[string]interface{}{"Branch": protectedBranch},
		})
		fmt.Println(msg)
	}

	if len(branchesToDelete) == 0 {
		msg, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "NoBranchesSelected"})
		fmt.Println(msg)
		os.Exit(0)
	}

	// Display confirmation
	confirmMsg, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "ConfirmDeletion"})
	fmt.Printf("\n%s\n", confirmMsg)

	branchHeader, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "Branch"})
	remoteHeader, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "Remote"})

	fmt.Printf("%-40s %s\n", branchHeader, remoteHeader)
	fmt.Println(strings.Repeat("-", 60))

	for _, branch := range branchesToDelete {
		parts := strings.SplitN(branch, "/", 2)
		if len(parts) == 2 {
			fmt.Printf("%-40s %s\n", parts[1], parts[0])
		} else {
			fmt.Printf("%-40s %s\n", branch, "(unknown)")
		}
	}
	fmt.Println(strings.Repeat("-", 60))

	// Use survey.Confirm for final confirmation
	confirmPrompt := &survey.Confirm{
		Message: "Proceed with deletion?",
		Default: false,
	}
	var confirm bool
	survey.AskOne(confirmPrompt, &confirm)

	if !confirm {
		cancelMsg, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: "DeletionCancelled"})
		fmt.Println(cancelMsg)
		os.Exit(0)
	}

	// Proceed with deletion
	for _, branch := range branchesToDelete {
		parts := strings.SplitN(branch, "/", 2)
		if len(parts) != 2 {
			fmt.Printf("Skipping invalid branch format: %s\n", branch)
			continue
		}
		remoteName := parts[0]
		branchName := parts[1]

		deleteCmd := exec.Command("git", "push", remoteName, "--delete", branchName)
		deleteOutput, err := deleteCmd.CombinedOutput()
		if err != nil {
			msg, _ := localizer.Localize(&i18n.LocalizeConfig{
				MessageID: "ErrorDeletingBranch",
				TemplateData: map[string]interface{}{"Branch": branch, "Error": err},
			})
			fmt.Println(msg)
			fmt.Println(string(deleteOutput))
		} else {
			msg, _ := localizer.Localize(&i18n.LocalizeConfig{
				MessageID: "BranchDeletedSuccessfully",
				TemplateData: map[string]interface{}{"Branch": branch},
			})
			fmt.Println(msg)
			fmt.Println(string(deleteOutput))
		}
	}
}
