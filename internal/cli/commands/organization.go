package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/kutbudev/ramorie-cli/internal/api"
	apierrors "github.com/kutbudev/ramorie-cli/internal/errors"
	"github.com/urfave/cli/v2"
)

// NewOrganizationCommand creates all subcommands for the 'organization' command group.
func NewOrganizationCommand() *cli.Command {
	return &cli.Command{
		Name:    "organization",
		Aliases: []string{"org"},
		Usage:   "Manage organizations",
		Subcommands: []*cli.Command{
			orgListCmd(),
			orgCreateCmd(),
			orgShowCmd(),
			// orgSwitchCmd(),    // TODO: Backend needs active org support
			// orgInviteCmd(),    // TODO: Backend needs member management
			// orgMembersCmd(),   // TODO: Backend needs member listing
			// orgLeaveCmd(),     // TODO: Backend needs leave functionality
		},
	}
}

// orgListCmd lists all organizations for the user.
func orgListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List your organizations",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			orgs, err := client.ListOrganizations()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(orgs) == 0 {
				fmt.Println("No organizations found. Use 'ramorie org create' to create one.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tROLE")
			fmt.Fprintln(w, "--\t----\t-----------\t----")

			for _, org := range orgs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					org.ID[:8],
					truncateString(org.Name, 30),
					truncateString(org.Description, 40),
					"owner") // TODO: Backend should return role when member system is added
			}
			w.Flush()
			return nil
		},
	}
}

// orgCreateCmd creates a new organization.
func orgCreateCmd() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a new organization",
		ArgsUsage: "[name]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Organization description",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization name is required")
			}
			name := c.Args().First()
			description := c.String("description")

			client := api.NewClient()
			org, err := client.CreateOrganization(name, description)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Organization '%s' created successfully!\n", org.Name)
			fmt.Printf("   ID: %s\n", org.ID[:8])
			if org.Description != "" {
				fmt.Printf("   Description: %s\n", org.Description)
			}
			return nil
		},
	}
}

// orgShowCmd shows details for a specific organization.
func orgShowCmd() *cli.Command {
	return &cli.Command{
		Name:      "show",
		Aliases:   []string{"info"},
		Usage:     "Show details for an organization",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required")
			}
			orgID := c.Args().First()

			client := api.NewClient()

			// If short ID provided, resolve to full ID
			if len(orgID) < 36 {
				orgs, err := client.ListOrganizations()
				if err != nil {
					fmt.Println(apierrors.ParseAPIError(err))
					return err
				}
				for _, org := range orgs {
					if strings.HasPrefix(org.ID, orgID) {
						orgID = org.ID
						break
					}
				}
			}

			org, err := client.GetOrganization(orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("Organization Details: %s\n", org.Name)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("ID:          %s\n", org.ID)
			fmt.Printf("Name:        %s\n", org.Name)
			fmt.Printf("Description: %s\n", org.Description)
			fmt.Printf("Owner ID:    %s\n", org.OwnerID[:8])
			fmt.Printf("Created At:  %s\n", org.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated At:  %s\n", org.UpdatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}

// orgSwitchCmd switches the active organization.
// TODO: Implement when backend supports active organization
/*
func orgSwitchCmd() *cli.Command {
	return &cli.Command{
		Name:      "switch",
		Aliases:   []string{"use"},
		Usage:     "Switch to a different organization",
		ArgsUsage: "[org-id]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("organization ID is required")
			}
			orgID := c.Args().First()

			client := api.NewClient()

			// Resolve short ID to full ID if needed
			if len(orgID) < 36 {
				orgs, err := client.ListOrganizations()
				if err != nil {
					fmt.Println(apierrors.ParseAPIError(err))
					return err
				}
				for _, org := range orgs {
					if strings.HasPrefix(org.ID, orgID) {
						orgID = org.ID
						break
					}
				}
			}

			err := client.SwitchOrganization(orgID)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Switched to organization %s\n", orgID[:8])
			return nil
		},
	}
}
*/

// orgInviteCmd invites a member to the organization.
// TODO: Implement when backend supports member management
/*
func orgInviteCmd() *cli.Command {
	return &cli.Command{
		Name:      "invite",
		Usage:     "Invite a member to the organization",
		ArgsUsage: "[email]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "role",
				Aliases: []string{"r"},
				Usage:   "Role for the member (member, admin)",
				Value:   "member",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("email address is required")
			}
			email := c.Args().First()
			role := c.String("role")

			client := api.NewClient()
			err := client.InviteOrganizationMember(email, role)
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Printf("✅ Invitation sent to %s with role: %s\n", email, role)
			return nil
		},
	}
}
*/

// orgMembersCmd lists organization members.
// TODO: Implement when backend supports member listing
/*
func orgMembersCmd() *cli.Command {
	return &cli.Command{
		Name:    "members",
		Aliases: []string{"ls-members"},
		Usage:   "List organization members",
		Action: func(c *cli.Context) error {
			client := api.NewClient()
			members, err := client.ListOrganizationMembers()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			if len(members) == 0 {
				fmt.Println("No members found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "EMAIL\tROLE\tJOINED")
			fmt.Fprintln(w, "-----\t----\t------")

			for _, member := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					member.Email,
					member.Role,
					member.JoinedAt.Format("2006-01-02"))
			}
			w.Flush()
			return nil
		},
	}
}
*/

// orgLeaveCmd leaves the organization.
// TODO: Implement when backend supports leave functionality
/*
func orgLeaveCmd() *cli.Command {
	return &cli.Command{
		Name:  "leave",
		Usage: "Leave the organization",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force leave without confirmation",
			},
		},
		Action: func(c *cli.Context) error {
			force := c.Bool("force")

			if !force {
				fmt.Print("Are you sure you want to leave the organization? (yes/no): ")
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "yes" {
					fmt.Println("Operation cancelled.")
					return nil
				}
			}

			client := api.NewClient()
			err := client.LeaveOrganization()
			if err != nil {
				fmt.Println(apierrors.ParseAPIError(err))
				return err
			}

			fmt.Println("✅ You have left the organization.")
			return nil
		},
	}
}
*/
