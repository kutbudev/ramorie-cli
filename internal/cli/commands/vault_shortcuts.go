package commands

import "github.com/urfave/cli/v2"

// NewUnlockCommand is a top-level shortcut for `ramorie setup vault unlock`.
// Same handler, less typing — vault unlock is invoked often enough to deserve
// a one-word command.
func NewUnlockCommand() *cli.Command {
	return &cli.Command{
		Name:  "unlock",
		Usage: "Unlock the encrypted vault (alias of `setup vault unlock`)",
		Action: func(c *cli.Context) error {
			return handleVaultUnlock()
		},
	}
}

// NewLockCommand is a top-level shortcut for `ramorie setup vault lock`.
func NewLockCommand() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "Lock the encrypted vault (alias of `setup vault lock`)",
		Action: func(c *cli.Context) error {
			return handleVaultLock()
		},
	}
}
