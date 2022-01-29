package main

import "flag"

// parameters of the app configuration to be read from the program exec command.
type prmAppConfig struct {
	// target of the filepath to config file
	filepath *string
}

// parses command-line arguments, and reads parameters of the app config from them.
func (x *prmAppConfig) read() {
	flag.StringVar(x.filepath, "config", "", "Filepath to JSON or YAML config file")

	flag.Parse()
}

func (x *prmAppConfig) filepathTo(dst *string) {
	x.filepath = dst
}
