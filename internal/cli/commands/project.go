package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/terzigolu/josepshbrain-go/config"
)

// Project is a simplified local struct to hold project data from the API.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NewProjectCmd creates the project command with subcommands, now fully API-driven.
func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project",
		Short:   "Project management commands",
		Aliases: []string{"proj"},
	}

	cmd.AddCommand(newProjectInitCmd())
	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectUseCmd())

	return cmd
}

// project init
func newProjectInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new project via API",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			payload := map[string]string{"name": projectName}
			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("Failed to create JSON payload: %v", err)
			}

			cfg, err := config.LoadCliConfig()
			if err != nil {
				log.Fatalf("Failed to load CLI config: %v", err)
			}
			apiURL := fmt.Sprintf("%s/v1/projects", cfg.ApiURL)

			resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Fatalf("Failed to create project via API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				log.Fatalf("API returned a non-201 status code: %s", resp.Status)
			}

			var newProject Project
			if err := json.NewDecoder(resp.Body).Decode(&newProject); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			fmt.Printf("✅ Project '%s' initialized successfully via API.\n", newProject.Name)

			// Automatically 'use' the new project
			cfg.ActiveProjectID = newProject.ID
			if err := config.SaveCliConfig(cfg); err != nil {
				log.Fatalf("Failed to set new project as active: %v", err)
			}
			fmt.Printf("👉 Switched to new project '%s'.\n", newProject.Name)
		},
	}
	return cmd
}

// project list
func newProjectListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all projects from the API and show the active one",
		Aliases: []string{"ls"},
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadCliConfig()
			if err != nil {
				log.Fatalf("Failed to load CLI config: %v", err)
			}
			apiURL := fmt.Sprintf("%s/v1/projects", cfg.ApiURL)

			resp, err := http.Get(apiURL)
			if err != nil {
				log.Fatalf("Failed to fetch projects from API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			var projects []Project
			if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			if len(projects) == 0 {
				fmt.Println("No projects found. Use 'jbraincli project init <name>' to create one.")
				return
			}

			fmt.Println("🏢 Available Projects:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tACTIVE")
			fmt.Fprintln(w, "--\t----\t-----------\t------")
			for _, p := range projects {
				activeMarker := ""
				if p.ID == cfg.ActiveProjectID {
					activeMarker = "➤"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", truncateString(p.ID, 8), p.Name, p.Description, activeMarker)
			}
			w.Flush()
		},
	}
	return cmd
}

// project use
func newProjectUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [name or ID prefix]",
		Short: "Set the active project",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			searchTerm := args[0]

			cfg, err := config.LoadCliConfig()
			if err != nil {
				log.Fatalf("Failed to load CLI config: %v", err)
			}
			apiURL := fmt.Sprintf("%s/v1/projects", cfg.ApiURL)

			resp, err := http.Get(apiURL)
			if err != nil {
				log.Fatalf("Failed to fetch projects from API: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("API returned a non-200 status code: %s", resp.Status)
			}

			var projects []Project
			if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
				log.Fatalf("Failed to decode API response: %v", err)
			}

			var foundProject *Project
			for i, p := range projects {
				if p.Name == searchTerm || strings.HasPrefix(p.ID, searchTerm) {
					foundProject = &projects[i]
					break
				}
			}

			if foundProject == nil {
				log.Fatalf("Project '%s' not found.", searchTerm)
			}

			cfg.ActiveProjectID = foundProject.ID
			if err := config.SaveCliConfig(cfg); err != nil {
				log.Fatalf("Failed to save configuration: %v", err)
			}

			fmt.Printf("✅ Switched to project '%s'.\n", foundProject.Name)
		},
	}
	return cmd
}


