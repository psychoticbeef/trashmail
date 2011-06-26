package main

import (
	"io/ioutil"
	"os"
	"bufio"
	"strings"
	"fmt"
	"regexp"
	"flag"
	"github.com/kless/goconfig/config"
)

var (
	home = os.ShellExpand("$HOME")

	c, _            = config.ReadDefault(home + "/.trashmailrc")
	host, _         = c.String("default", "host")
	destination, _  = c.String("default", "destination")
	mailListFile, _ = c.String("default", "maillist")

	// \\w doesn't work "illegal backslash escape", greedy matching doesn't work "repeated closure"
	// this forced me to produce such a fucked up, and limited expression :/
	// ^ and $ seem to be b0rked aswell m(
	patternOriginalTo = regexp.MustCompile("X-Original-To: ([A-Za-z]+@[A-Za-z]+\\.[A-Za-z]+)")
	patternEmail      = regexp.MustCompile("[A-Za-z]+@" + host)
	patternSubject    = regexp.MustCompile("Subject: ")

	alter = flag.Bool("a", false, "change email headers, not required for condition checking")

	// email address is key, corresponding service value
	service = make(map[string]string, 1)
)


func main() {
	flag.Parse()

	// read file that contains email addresses and corresponding services
	f, _ := os.Open(os.ShellExpand(mailListFile))
	defer f.Close()
	r := bufio.NewReader(f)
	kvLine, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(kvLine)
		kv := strings.Split(s, "\t", -1)
		service[kv[0]] = kv[1]
		kvLine, isPrefix, err = r.ReadLine()
	}

	// we get the email headers from procmail via stdin
	stdin, _ := ioutil.ReadAll(os.Stdin)
	mailHeader := string(stdin)

	matches := patternOriginalTo.FindAllStringSubmatch(mailHeader, -1)

	if len(matches) != 1 {
		os.Exit(1)
	}
	// recipient email address from "X-Original-To" field
	recipient := strings.ToLower(string(matches[0][1]))

	// check if the service is in .maillist
	serviceName, ok := service[recipient]
	if !ok {
		os.Exit(1)
	}

	// shall we alter the header content? not required for condition check
	if !*alter {
		os.Exit(0)
	}

	// change all occurrences of an email address at our domain to one standard email address
	// if this isn't done, mobileme has trouble filtering m(
	mailHeader = patternEmail.ReplaceAllString(mailHeader, destination)
	// add service name to the subject line
	mailHeader = patternSubject.ReplaceAllString(mailHeader, "Subject: ["+serviceName+"] ")

	fmt.Print(mailHeader)
}
