package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `hgy

Usage:
    hgy new
    hgy --help

Options:
    -h --help  Sow this screen
`

	arguments, error := docopt.Parse(usage, nil, true, "Hgy 0.01", false)

	fmt.Println()
	fmt.Println(arguments)
	fmt.Println(error)
}
