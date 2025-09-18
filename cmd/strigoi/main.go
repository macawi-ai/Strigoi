// Strigoi Security Validation Platform
// Copyright © 2025 Macawi LLC. All Rights Reserved.
// Licensed under CC BY-NC-SA 4.0: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"os"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
