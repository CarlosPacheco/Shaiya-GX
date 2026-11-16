package control

import "os"

// Windows detached processes currently use taskkill in Stop.
func gracefulStop(process *os.Process) { process.Kill() }
