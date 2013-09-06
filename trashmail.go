package main

import (
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/jonsen/goconfig/config"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"net/url"
)

var (
	home = os.ExpandEnv("$HOME")

	c, _           = config.ReadDefault(home + "/.trashmailrc")
	host, _        = c.String("default", "host")
	destination, _ = c.String("default", "destination")

	// \\w doesn't work "illegal backslash escape", greedy matching doesn't work "repeated closure"
	// this forced me to produce such a fucked up, and limited expression :/
	// ^ and $ seem to be b0rked aswell m(
	patternOriginalTo = regexp.MustCompile("X-Original-To: ([A-Za-z]+@[A-Za-z]+\\.[A-Za-z]+)")
	patternEmail      = regexp.MustCompile("[A-Za-z]+@" + host)
	patternSubject    = regexp.MustCompile("Subject: ")

	alter = flag.Bool("a", false, "change email headers, not required for condition checking")

	db redis.Conn
)

const (
	Sforward = 1 << iota
	Sprowl
	Sspam
	Smaildir
)

func main() {
	flag.Parse()

	// we get the email headers from procmail via stdin
	stdin, _ := ioutil.ReadAll(os.Stdin)
	mailHeader := string(stdin)

	matches := patternOriginalTo.FindAllStringSubmatch(mailHeader, -1)
	full_subject := regexp.MustCompile("(?m)^Subject: .*?$").FindString(mailHeader)
	if len(full_subject) > 9 {
		full_subject = full_subject[9:len(full_subject)-1]
	} else {
		full_subject = ""
	}

	if len(matches) != 1 {
		os.Exit(1)
	}
	// recipient email address from "X-Original-To" field
	recipient := strings.ToLower(string(matches[0][1]))

	// check if the service is in .maillist
	db, _ := redis.Dial("tcp", ":6379")
	defer db.Close()
	id, err := redis.Int(db.Do("get", recipient))
	if err != nil {
		db.Do("incr", "unknown_deleted")
		os.Exit(1)
	}

	db_prefix := "m" + strconv.Itoa(id) + ":"

	serviceName, err := redis.String(db.Do("get", db_prefix+"service"))

	state, err := redis.Int(db.Do("get", db_prefix+"state"))

	// only do the db stuff, if we do not need to alter
	// prowling faster than emailing? lelz
	if state&Sprowl != 0  && !*alter {
		db.Do("incr", db_prefix+"prowld")
		db.Do("incr", "total_prowled")
		user_id, _ := redis.Int(db.Do("get", db_prefix+"user"))
		user_prefix := "u" + strconv.Itoa(user_id) + ":"
		apikey, _ := redis.String(db.Do("get", user_prefix+"prowl"))
		apikey = url.QueryEscape(apikey)
		event := url.QueryEscape(serviceName)
		description := url.QueryEscape(full_subject)
		prowl := fmt.Sprintf("apikey=%s&priority=0&application=trashmail&event=%s&description=%s", apikey, event, description)
		query := "https://prowl.weks.net/publicapi/add?" + prowl
		http.Get(query)
	}

	if state&Sforward == 0 && !*alter {
		db.Do("incr", db_prefix+"rejected")
		db.Do("incr", "total_rejected")
		os.Exit(1)
	}


	// shall we alter the header content? not required for condition check
	if !*alter {
		fmt.Print("/home/da/Maildir/")
		os.Exit(0)
	}

	db.Do("incr", db_prefix+"forwarded")
	db.Do("set", db_prefix+"last", time.Now().Unix())
	db.Do("incr", "total_forwarded")

	// change all occurrences of an email address at our domain to one standard email address
	// if this isn't done, mobileme has trouble filtering m(
	mailHeader = patternEmail.ReplaceAllString(mailHeader, destination)
	// add service name to the subject line
	mailHeader = patternSubject.ReplaceAllString(mailHeader, "Subject: ["+serviceName+"] ")

	fmt.Print(mailHeader)
}
