package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/api"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/urfave/cli/v2"
)

// NewPlanCommand creates all subcommands for the 'plan' command group.
func NewPlanCommand() *cli.Command {
	return &cli.Command{
		Name:    "plan",
		Aliases: []string{"pl"},
		Usage:   "Multi-agent AI planning with consensus-based execution",
		Subcommands: []*cli.Command{
			planCreateCmd(),
			planListCmd(),
			planStatusCmd(),
			planApplyCmd(),
			planCancelCmd(),
			planResumeCmd(),
			planOpenCmd(),
			planArtifactsCmd(),
			planRisksCmd(),
		},
	}
}

// planCreateCmd creates a new plan run
func planCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Aliases:   []string{"new"},
		Usage:     "Create and start a new planning run",
		ArgsUsage: "[goal/requirements]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "Plan title (defaults to first 50 chars of requirements)"},
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Target project ID or name"},
			&cli.StringFlag{Name: "type", Value: "feature", Usage: "Plan type: feature, refactor, bugfix, research"},
			&cli.Float64Flag{Name: "consensus", Value: 0.75, Usage: "Consensus threshold (0.0-1.0)"},
			&cli.IntFlag{Name: "agents", Value: 3, Usage: "Number of proposal agents per phase"},
			&cli.Float64Flag{Name: "budget", Value: 5.0, Usage: "Max budget in USD"},
			&cli.BoolFlag{Name: "no-stream", Usage: "Disable streaming output"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Preview configuration without executing"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("goal/requirements are required")
			}
			requirements := strings.Join(c.Args().Slice(), " ")

			title := c.String("title")
			if title == "" {
				title = requirements
				if len(title) > 50 {
					title = title[:47] + "..."
				}
			}

			planType := c.String("type")
			consensusThreshold := c.Float64("consensus")
			agentCount := c.Int("agents")
			budgetUSD := c.Float64("budget")
			dryRun := c.Bool("dry-run")

			// Validate plan type
			validTypes := []string{"feature", "refactor", "bugfix", "research"}
			typeValid := false
			for _, t := range validTypes {
				if planType == t {
					typeValid = true
					break
				}
			}
			if !typeValid {
				return fmt.Errorf("invalid plan type '%s'. Must be one of: %s", planType, strings.Join(validTypes, ", "))
			}

			// Resolve project
			client := api.NewClient()
			var projectID string
			projectArg := c.String("project")
			if projectArg != "" {
				projects, err := client.ListProjects()
				if err != nil {
					fmt.Println(apierrors.ParseAPIError(err))
					return err
				}
				for _, p := range projects {
					if p.ID.String() == projectArg ||
						strings.HasPrefix(p.ID.String(), projectArg) ||
						strings.EqualFold(p.Name, projectArg) {
						projectID = p.ID.String()
						break
					}
				}
				if projectID == "" {
					return fmt.Errorf("project '%s' not found", projectArg)
				}
			}

			if dryRun {
				fmt.Println("🔍 Dry Run - Plan Configuration:")
				fmt.Println(strings.Repeat("-", 50))
				fmt.Printf("Title:               %s\n", title)
				fmt.Printf("Type:                %s\n", planType)
				fmt.Printf("Requirements:        %s\n", truncateString(requirements, 100))
				fmt.Printf("Consensus Threshold: %.2f\n", consensusThreshold)
				fmt.Printf("Proposal Agents:     %d\n", agentCount)
				fmt.Printf("Budget (USD):        $%.2f\n", budgetUSD)
				if projectID != "" {
					fmt.Printf("Project ID:          %s\n", projectID[:8])
				}
				fmt.Println("\n✅ Configuration valid. Remove --dry-run to execute.")
				return nil
			}

			// Create plan
			plan, err := client.CreatePlan(api.CreatePlanRequest{
				Title:        title,
				Requirements: requirements,
				ProjectID:    projectID,
				Type:         planType,
				Configuration: &api.PlanConfiguration{
					ConsensusThreshold: consensusThreshold,
					ProposalAgentCount: agentCount,
					BudgetUSD:          budgetUSD,
				},
			})
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Plan '%s' created successfully!\n", plan.Title)
			fmt.Printf("   ID: %s\n", plan.ID[:8])
			fmt.Printf("   Type: %s\n", plan.Type)
			fmt.Printf("   Status: %s\n", plan.Status)
			fmt.Println("\n💡 Use 'ramorie plan status " + plan.ID[:8] + "' to check progress")
			fmt.Println("💡 Use 'ramorie plan open " + plan.ID[:8] + "' to view in browser")

			return nil
		},
	}
}

// planListCmd lists plan runs
func planListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List plan runs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "status", Aliases: []string{"s"}, Usage: "Filter by status (pending, running, completed, failed, cancelled)"},
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "Filter by type (feature, refactor, bugfix, research)"},
			&cli.StringFlag{Name: "project", Aliases: []string{"p"}, Usage: "Filter by project ID or name"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"l"}, Value: 20, Usage: "Limit results"},
		},
		Action: func(c *cli.Context) error {
			status := c.String("status")
			planType := c.String("type")
			limit := c.Int("limit")

			client := api.NewClient()

			// Resolve project if provided
			var projectID string
			projectArg := c.String("project")
			if projectArg != "" {
				projects, err := client.ListProjects()
				if err == nil {
					for _, p := range projects {
						if p.ID.String() == projectArg ||
							strings.HasPrefix(p.ID.String(), projectArg) ||
							strings.EqualFold(p.Name, projectArg) {
							projectID = p.ID.String()
							break
						}
					}
				}
			}

			plans, err := client.ListPlans(api.ListPlansFilter{
				Status:    status,
				Type:      planType,
				ProjectID: projectID,
				Limit:     limit,
			})
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(plans) == 0 {
				fmt.Println("No plans found. Use 'ramorie plan create \"goal\"' to start one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTITLE\tTYPE\tSTATUS\tPROGRESS\tCREATED")
			fmt.Fprintln(w, "--\t-----\t----\t------\t--------\t-------")

			for _, p := range plans {
				statusIcon := getStatusIcon(p.Status)
				createdAt := p.CreatedAt.Format("Jan 02 15:04")
				fmt.Fprintf(w, "%s\t%s\t%s\t%s %s\t%d%%\t%s\n",
					p.ID[:8],
					truncateString(p.Title, 35),
					p.Type,
					statusIcon,
					p.Status,
					p.Progress,
					createdAt)
			}
			w.Flush()
			return nil
		},
	}
}

// planStatusCmd shows detailed status of a plan
func planStatusCmd() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Aliases:   []string{"show", "info"},
		Usage:     "Show detailed status of a plan run",
		ArgsUsage: "[plan-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			client := api.NewClient()
			plan, err := client.GetPlan(planID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			statusIcon := getStatusIcon(plan.Status)
			fmt.Printf("%s Plan: %s\n", statusIcon, plan.Title)
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("ID:            %s\n", plan.ID)
			fmt.Printf("Type:          %s\n", plan.Type)
			fmt.Printf("Status:        %s\n", plan.Status)
			fmt.Printf("Progress:      %d%%\n", plan.Progress)
			fmt.Printf("Current Phase: %s\n", plan.CurrentPhase)
			fmt.Printf("Created:       %s\n", plan.CreatedAt.Format("2006-01-02 15:04:05"))

			if plan.StartedAt != nil {
				fmt.Printf("Started:       %s\n", plan.StartedAt.Format("2006-01-02 15:04:05"))
			}
			if plan.CompletedAt != nil {
				fmt.Printf("Completed:     %s\n", plan.CompletedAt.Format("2006-01-02 15:04:05"))
			}

			// Budget info
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("Tokens Used:   %d\n", plan.TokensUsed)
			fmt.Printf("Cost (USD):    $%.4f\n", plan.SpentBudgetUSD)
			if plan.MaxBudgetUSD != nil {
				fmt.Printf("Budget (USD):  $%.2f\n", *plan.MaxBudgetUSD)
			}

			// Requirements
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("Requirements:\n%s\n", plan.Requirements)

			// Phases if available
			if len(plan.Phases) > 0 {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Println("Phases:")
				for _, phase := range plan.Phases {
					phaseIcon := getStatusIcon(phase.Status)
					consensusStr := ""
					if phase.ConsensusScore != nil {
						consensusStr = fmt.Sprintf(" (%.0f%% consensus)", *phase.ConsensusScore*100)
					}
					fmt.Printf("  %s %d. %s - %s%s\n",
						phaseIcon,
						phase.Sequence,
						phase.Phase,
						phase.Status,
						consensusStr)
				}
			}

			// Results if completed
			if plan.Status == "completed" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Printf("Results:       %d tasks, %d ADRs, %d risks\n",
					plan.TaskCount, plan.ADRCount, plan.RiskCount)
				if plan.FinalConsensusScore != nil {
					fmt.Printf("Consensus:     %.0f%%\n", *plan.FinalConsensusScore*100)
				}
				fmt.Println("\n💡 Use 'ramorie plan apply " + plan.ID[:8] + "' to create tasks and ADRs")
			}

			// Error if failed
			if plan.Error != "" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Printf("⚠️  Error: %s\n", plan.Error)
				fmt.Println("\n💡 Use 'ramorie plan resume " + plan.ID[:8] + "' to retry")
			}

			return nil
		},
	}
}

// planApplyCmd applies plan results (creates tasks/ADRs)
func planApplyCmd() *cli.Command {
	return &cli.Command{
		Name:      "apply",
		Usage:     "Apply plan results - create tasks and ADRs",
		ArgsUsage: "[plan-id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "tasks-only", Usage: "Only create tasks, skip ADRs"},
			&cli.BoolFlag{Name: "adrs-only", Usage: "Only create ADRs, skip tasks"},
			&cli.StringFlag{Name: "task-status", Value: "TODO", Usage: "Initial status for created tasks"},
			&cli.StringFlag{Name: "adr-status", Value: "draft", Usage: "Initial status for created ADRs"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Preview what would be created"},
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Apply without confirmation"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			tasksOnly := c.Bool("tasks-only")
			adrsOnly := c.Bool("adrs-only")
			taskStatus := c.String("task-status")
			adrStatus := c.String("adr-status")
			dryRun := c.Bool("dry-run")
			force := c.Bool("force")

			client := api.NewClient()

			// Get plan first to show what will be created
			plan, err := client.GetPlan(planID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if plan.Status != "completed" {
				return fmt.Errorf("plan must be completed before applying. Current status: %s", plan.Status)
			}

			applyTasks := !adrsOnly
			applyADRs := !tasksOnly

			fmt.Printf("📋 Plan: %s\n", plan.Title)
			fmt.Println(strings.Repeat("-", 50))
			if applyTasks {
				fmt.Printf("Tasks to create: %d (status: %s)\n", plan.TaskCount, taskStatus)
			}
			if applyADRs {
				fmt.Printf("ADRs to create:  %d (status: %s)\n", plan.ADRCount, adrStatus)
			}

			if dryRun {
				fmt.Println("\n🔍 Dry run - no changes made")
				return nil
			}

			// Confirmation
			if !force {
				fmt.Print("\nProceed? (y/N): ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Apply
			result, err := client.ApplyPlan(planID, api.ApplyPlanRequest{
				ApplyTasks: applyTasks,
				ApplyADRs:  applyADRs,
				TaskStatus: taskStatus,
				ADRStatus:  adrStatus,
			})
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("\n✅ Plan applied successfully!\n")
			if result.TasksCreated > 0 {
				fmt.Printf("   Tasks created: %d\n", result.TasksCreated)
			}
			if result.ADRsCreated > 0 {
				fmt.Printf("   ADRs created:  %d\n", result.ADRsCreated)
			}

			return nil
		},
	}
}

// planCancelCmd cancels a running plan
func planCancelCmd() *cli.Command {
	return &cli.Command{
		Name:      "cancel",
		Usage:     "Cancel a running plan",
		ArgsUsage: "[plan-id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Cancel without confirmation"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			if !c.Bool("force") {
				fmt.Printf("Are you sure you want to cancel plan %s? (y/N): ", planID[:8])
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			client := api.NewClient()
			if err := client.CancelPlan(planID); err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("🛑 Plan %s cancelled.\n", planID[:8])
			fmt.Println("💡 Use 'ramorie plan resume " + planID[:8] + "' to continue later")
			return nil
		},
	}
}

// planResumeCmd resumes a failed/cancelled plan
func planResumeCmd() *cli.Command {
	return &cli.Command{
		Name:      "resume",
		Usage:     "Resume a failed or cancelled plan",
		ArgsUsage: "[plan-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			client := api.NewClient()
			plan, err := client.ResumePlan(planID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("▶️  Plan '%s' resumed.\n", plan.Title)
			fmt.Printf("   Status: %s\n", plan.Status)
			fmt.Printf("   Current Phase: %s\n", plan.CurrentPhase)
			fmt.Println("\n💡 Use 'ramorie plan status " + planID[:8] + "' to check progress")
			return nil
		},
	}
}

// planOpenCmd opens plan in browser
func planOpenCmd() *cli.Command {
	return &cli.Command{
		Name:      "open",
		Usage:     "Open plan in web dashboard",
		ArgsUsage: "[plan-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			// Expand short ID if needed
			client := api.NewClient()
			plan, err := client.GetPlan(planID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			url := fmt.Sprintf("https://app.ramorie.com/planner/%s", plan.ID)
			fmt.Printf("🌐 Opening %s\n", url)

			// Open in browser
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", url)
			case "linux":
				cmd = exec.Command("xdg-open", url)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", url)
			default:
				fmt.Printf("Please open: %s\n", url)
				return nil
			}

			return cmd.Start()
		},
	}
}

// planArtifactsCmd lists plan artifacts
func planArtifactsCmd() *cli.Command {
	return &cli.Command{
		Name:      "artifacts",
		Aliases:   []string{"art"},
		Usage:     "List or view plan artifacts",
		ArgsUsage: "[plan-id] [artifact-type]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: "Artifact type: plan_md, plan_json, tasks_json, adrs_json, risks_json"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			client := api.NewClient()

			// Get specific artifact type
			artifactType := c.String("type")
			if artifactType == "" && c.NArg() > 1 {
				artifactType = c.Args().Get(1)
			}

			if artifactType != "" {
				artifact, err := client.GetPlanArtifact(planID, artifactType)
				if err != nil {
					fmt.Println(apierrors.ParseAPIError(err))
					return err
				}

				fmt.Printf("📄 Artifact: %s (%s)\n", artifact.Name, artifact.Type)
				fmt.Println(strings.Repeat("-", 60))
				fmt.Println(artifact.Content)
				return nil
			}

			// List all artifacts
			artifacts, err := client.ListPlanArtifacts(planID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(artifacts) == 0 {
				fmt.Println("No artifacts found. Plan may not be completed yet.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TYPE\tNAME\tSIZE\tCREATED")
			fmt.Fprintln(w, "----\t----\t----\t-------")

			for _, a := range artifacts {
				createdAt := a.CreatedAt.Format("Jan 02 15:04")
				fmt.Fprintf(w, "%s\t%s\t%d bytes\t%s\n",
					a.Type,
					a.Name,
					len(a.Content),
					createdAt)
			}
			w.Flush()

			fmt.Println("\n💡 Use 'ramorie plan artifacts " + planID[:8] + " --type plan_md' to view specific artifact")
			return nil
		},
	}
}

// planRisksCmd shows plan risks
func planRisksCmd() *cli.Command {
	return &cli.Command{
		Name:      "risks",
		Usage:     "Show identified risks for a plan",
		ArgsUsage: "[plan-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("plan ID is required")
			}
			planID := c.Args().First()

			client := api.NewClient()
			risks, err := client.ListPlanRisks(planID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(risks) == 0 {
				fmt.Println("No risks identified. Plan may not be completed yet.")
				return nil
			}

			fmt.Printf("⚠️  Risks (%d identified)\n", len(risks))
			fmt.Println(strings.Repeat("-", 60))

			for i, r := range risks {
				severityIcon := getSeverityIcon(r.Severity)
				fmt.Printf("\n%d. %s [%s] %s\n", i+1, severityIcon, r.Severity, r.Title)
				fmt.Printf("   Category:   %s\n", r.Category)
				fmt.Printf("   Status:     %s\n", r.Status)
				if r.Description != "" {
					fmt.Printf("   Description: %s\n", truncateString(r.Description, 80))
				}
				if r.Mitigation != "" {
					fmt.Printf("   Mitigation: %s\n", truncateString(r.Mitigation, 80))
				}
			}

			return nil
		},
	}
}

// Helper functions

func getStatusIcon(status string) string {
	switch status {
	case "pending":
		return "⏳"
	case "running", "routing", "discovery", "defining", "developing", "delivering":
		return "🔄"
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "cancelled":
		return "🛑"
	default:
		return "❓"
	}
}

func getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

// Used for streaming (future enhancement)
func streamPlanProgress(client *api.Client, planID string) {
	// TODO: Implement SSE streaming for real-time progress
	fmt.Println("⏳ Plan execution started...")
	for {
		plan, err := client.GetPlan(planID)
		if err != nil {
			fmt.Printf("Error checking status: %v\n", err)
			return
		}

		fmt.Printf("\r🔄 Phase: %s | Progress: %d%%", plan.CurrentPhase, plan.Progress)

		if plan.Status == "completed" || plan.Status == "failed" || plan.Status == "cancelled" {
			fmt.Println()
			break
		}

		time.Sleep(2 * time.Second)
	}
}
