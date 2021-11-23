package main

import (
	"io"
	// no import scenarigo
)

var (
	String    = "string"
	Pointer   = &String
	Interface = io.Reader(nil)
)

func Function() string { return "function" }
