# GitHub Project Board Setup

Since GitHub Projects must be created through the web interface, here are the steps to set up the Kanban board:

## 1. Create the Project

1. Go to: https://github.com/macawi-ai/Strigoi
2. Click on "Projects" tab
3. Click "New project"
4. Choose "Board" template
5. Name it: "Strigoi Development"
6. Set visibility: Public

## 2. Configure Columns

Create these columns in order:
1. **Backlog** - New issues and ideas
2. **Ready** - Issues ready to work on
3. **In Progress** - Currently being worked on (limit: 3)
4. **In Review** - Code review or testing
5. **Done** - Completed work

## 3. Add Automation

For each column, set up automation:

### Backlog
- When: Issue opened → Move to Backlog
- When: Issue reopened → Move to Backlog

### In Progress
- When: PR opened → Move linked issues to In Progress
- Set WIP limit: 3 items

### In Review
- When: PR marked ready for review → Move to In Review

### Done
- When: Issue closed → Move to Done
- When: PR merged → Move to Done

## 4. Add Existing Issues

Add these issues to the board:
- #2: Implement real probe north module → Ready
- #3: Write unit tests for core REPL → Ready
- #4: Implement stream tap module → Backlog
- #5: Create architecture documentation → Ready

## 5. Create Views

### Development View (default)
- Group by: Status
- Filter: is:open

### Security Modules View
- Group by: Status
- Filter: label:module

### Sprint View
- Group by: Assignee
- Filter: is:open milestone:current

## 6. Set Project Settings

- Enable "Track project progress"
- Add README describing workflow
- Link to DEVELOPMENT_METHODOLOGY.md

## Workflow Rules

1. **Issue Creation**: All work starts with an issue
2. **WIP Limits**: Max 3 items in progress
3. **Definition of Ready**:
   - Clear acceptance criteria
   - Labeled appropriately
   - Estimated (if applicable)

4. **Definition of Done**:
   - Code complete
   - Tests written
   - Documentation updated
   - PR approved and merged

## Quick Links

After setup, the project will be available at:
https://github.com/macawi-ai/Strigoi/projects/1

This provides a clear visual workflow for managing Strigoi development.