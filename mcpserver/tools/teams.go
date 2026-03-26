// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost/server/public/model"
)

// CreateTeamArgs represents arguments for the create_team tool (dev mode only)
type CreateTeamArgs struct {
	Name        string `json:"name" jsonschema:"URL name for the team,minLength=1,maxLength=64"`
	DisplayName string `json:"display_name" jsonschema:"Display name for the team,minLength=1,maxLength=64"`
	Type        string `json:"type" jsonschema:"Team type,enum=O,enum=I"`
	Description string `json:"description" jsonschema:"Team description,maxLength=255"`
	TeamIcon    string `json:"team_icon,omitempty" access:"local" jsonschema:"File path or URL to set as team icon (supports .jpeg, .jpg, .png, .gif)"`
}

// AddUserToTeamArgs represents arguments for the add_user_to_team tool (dev mode only)
type AddUserToTeamArgs struct {
	UserID string `json:"user_id" jsonschema:"ID of the user to add"`
	TeamID string `json:"team_id" jsonschema:"ID of the team to add user to"`
}

// getTeamTools returns all team-related tools
func (p *MattermostToolProvider) getTeamTools() []MCPTool {
	return []MCPTool{}
}

// getDevTeamTools returns development team-related tools for MCP
// Disabled per configuration request; no tools returned.
func (p *MattermostToolProvider) getDevTeamTools() []MCPTool {
	return []MCPTool{}
}

// toolCreateTeam implements the create_team tool using the context client
func (p *MattermostToolProvider) toolCreateTeam(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args CreateTeamArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool create_team: %w", err)
	}

	// Validate required fields
	if args.Name == "" {
		return "name is required", fmt.Errorf("name cannot be empty")
	}
	if args.DisplayName == "" {
		return "display_name is required", fmt.Errorf("display_name cannot be empty")
	}
	if args.Type == "" {
		return "type is required", fmt.Errorf("type cannot be empty")
	}

	// Validate team type
	if args.Type != "O" && args.Type != "I" {
		return "type must be 'O' for open or 'I' for invite only", fmt.Errorf("invalid team type: %s", args.Type)
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	// Create the team
	team := &model.Team{
		Name:        args.Name,
		DisplayName: args.DisplayName,
		Type:        args.Type,
		Description: args.Description,
	}

	createdTeam, _, err := client.CreateTeam(ctx, team)
	if err != nil {
		return "failed to create team", fmt.Errorf("error creating team: %w", err)
	}

	var teamIconMessage string
	// Upload team icon if specified
	if args.TeamIcon != "" {
		// Validate image file type
		fileName := extractFileNameForLocal(args.TeamIcon, mcpContext.AccessMode)
		if !isValidImageFile(fileName) {
			teamIconMessage = " (team icon upload failed: unsupported file type, only .jpeg, .jpg, .png, .gif are supported)"
		} else {
			imageData, err := fetchFileDataForLocal(args.TeamIcon, mcpContext.AccessMode)
			if err != nil {
				teamIconMessage = fmt.Sprintf(" (team icon upload failed: %v)", err)
			} else {
				_, err = client.SetTeamIcon(ctx, createdTeam.Id, imageData)
				if err != nil {
					teamIconMessage = fmt.Sprintf(" (team icon upload failed: %v)", err)
				} else {
					teamIconMessage = " (team icon uploaded successfully)"
				}
			}
		}
	}

	return fmt.Sprintf("Successfully created team '%s' with ID: %s%s", createdTeam.DisplayName, createdTeam.Id, teamIconMessage), nil
}

// toolAddUserToTeam implements the add_user_to_team tool using the context client
func (p *MattermostToolProvider) toolAddUserToTeam(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args AddUserToTeamArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool add_user_to_team: %w", err)
	}

	// Validate required fields
	if !model.IsValidId(args.UserID) {
		return "invalid user_id format", fmt.Errorf("user_id must be a valid ID")
	}
	if !model.IsValidId(args.TeamID) {
		return "invalid team_id format", fmt.Errorf("team_id must be a valid ID")
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	// Add user to team
	_, _, err = client.AddTeamMember(ctx, args.TeamID, args.UserID)
	if err != nil {
		return "failed to add user to team", fmt.Errorf("error adding user to team: %w", err)
	}

	// Get user and team info for confirmation
	user, _, userErr := client.GetUser(ctx, args.UserID, "")
	team, _, teamErr := client.GetTeam(ctx, args.TeamID, "")

	if userErr != nil || teamErr != nil {
		return fmt.Sprintf("Successfully added user %s to team %s", args.UserID, args.TeamID), nil
	}

	return fmt.Sprintf("Successfully added user '%s' to team '%s'", user.Username, team.DisplayName), nil
}
