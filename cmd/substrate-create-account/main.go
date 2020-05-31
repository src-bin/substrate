package main

import "flag"

func main() {
	domain := flag.String("domain", "", "Domain for this new AWS account")
	environment := flag.String("environment", "", "Environment for this new AWS account")
	quality := flag.String("quality", "", "Quality for this new AWS account")
	flag.Parse()

}
