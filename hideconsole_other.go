//go:build !windows

package main

// Console hiding is a Windows-only concern (shell:startup launch).
func hideConsoleIfOwn() {}
