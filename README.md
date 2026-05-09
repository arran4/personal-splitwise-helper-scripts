# Personal Splitwise Helper Scripts

A set of command-line tools to interact with Splitwise.
It includes tools for managing expenses, groups, friends, tokens, and current user profile.

## Setup
It requires a Splitwise API token, which can be acquired on the splitwise portal, and stored either in the environment variable `SPLITWISE_API_TOKEN` or in `~/.config/splitwise/token`.

## Commands

### `expenses`
Manage your splitwise expenses. Contains various sub-commands:

* `list`
```text
Usage of list:
  -dated-after string
	Dated after
  -dated-before string
	Dated before
  -friend-id int
	Only expenses between current and provided user
  -group-id int
	Only expenses in that group will be returned
  -limit int
	Number of expenses to fetch per page (default 20)
  -offset int
	Initial offset
  -pages string
	Page selection: N, N-M, N-, or all
  -updated-after string
	Updated after
  -updated-before string
	Updated before
```

* `show`
```text
Usage of show:
  -id string
	ID of the expense to show
  -refresh
	Force refresh from API instead of using cache
```

* `edit`
```text
Usage of edit:
  -id string
	ID of the expense to edit
  -limit int
	Number of recent expenses to fetch per page when selecting (default 20)
  -offset int
	Initial offset when selecting a recent expense
  -pages string
	Page selection for recent expense chooser: N, N-M, N-, or all
  -refresh
	Force refresh from API instead of using cache
  -verbose
	Print the full server success payload after send
```

* `new`
```text
Usage of new:
  -friend-id int
	Create the expense with this friend
  -group-id int
	Create the expense in this group
  -verbose
	Print the full server success payload after send
```

* `import`
```text
Usage: expenses import <doordash|bunnings|amazon|woolworths|aussiebb> email text [--mode auto|new|update] [--stdin] [--id <expense-id>] [--group-id <id>|--friend-id <id>] [--payer <user-id>|--friend-paid]
```

### `friends`
Manage splitwise friends.
```text
Usage: friends <get|list>
```

### `groups`
Manage splitwise groups.
```text
Usage: groups <get|list>
```

### `token`
Manage splitwise API tokens.
```text
Usage: token <set|get|delete|test>
```

### `user`
Show user information.
```text
Usage: user <get|show>
```

## Features
- **Extensive Expenses Management:** You can quickly fetch, list, update, and manage your splitwise expenses directly from your terminal. Supports caching expenses.
- **TUI-Based Edit/New Views:** With the edit and new commands, a Terminal UI is provided making it very easy to manage split calculations, details, category, dates, groups, friends, etc.
- **Receipt Parsing (Imports):** Have a receipt from Amazon, AussieBB, Bunnings, Doordash, or Woolworths? Just pipe it into the `expenses import` command and it automatically parses the list of items for you.
- **Caching:** Automatically caches groups, friends, expenses, users, lowering the amount of API requests necessary.
- **Minimal Dependencies:** The core functions operate entirely with pure Go libraries and only a select few specialized TUI components to remain highly performant.

## Screenshots
### Commands Usage
![Commands Usage](docs/help.gif)


### Terminal UI (Edit / New / Mock)
![TUI Edit Expense](docs/edit.gif)

## Install

### GitHub Releases
Download binaries from: https://github.com/arran4/personal-splitwise-helper-scripts/releases

### Go install
go install github.com/arran4/personal-splitwise-helper-scripts/cmd/expenses@latest
go install github.com/arran4/personal-splitwise-helper-scripts/cmd/friends@latest
go install github.com/arran4/personal-splitwise-helper-scripts/cmd/groups@latest
go install github.com/arran4/personal-splitwise-helper-scripts/cmd/token@latest
go install github.com/arran4/personal-splitwise-helper-scripts/cmd/user@latest

### Native packages
- Debian/Ubuntu (`.deb`): see Releases assets
- RPM (`.rpm`): see Releases assets
- Alpine (`.apk`): see Releases assets
- Arch (`.pkg.tar.zst` or repo): see Releases assets
