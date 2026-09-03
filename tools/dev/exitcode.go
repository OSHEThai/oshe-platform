package main

// Exit codes form the stable wrapper contract. The meanings are fixed and must
// not be reordered.
const (
	ExitSuccess            = 0 // command completed without failure
	ExitWrappedToolFailure = 1 // a started runner exited with a nonzero status
	ExitUsage              = 2 // unknown command, flag, or missing argument
	ExitPrecondition       = 3 // governed root or required file absent/rejected
	ExitContractFailure    = 4 // a governed contract validation failed
	ExitEnvironment        = 5 // required runner/executable not present
)
