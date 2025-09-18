// Strigoi Security Validation Platform
// Copyright © 2025 Macawi LLC. All Rights Reserved.
// Licensed under AGPL-3.0: https://www.gnu.org/licenses/agpl-3.0.html
// Commercial licenses available at support@macawi.ai

package main

import (
	"os"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
