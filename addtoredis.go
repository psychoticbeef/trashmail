package main

import (
	//	"encoding/json"
	"bufio"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type Id struct {
	Email     string
	User      int
	State     int
	Filter    string
	Last      int64
	Forwarded int
	Rejected  int
	Service   string
}

type User struct {
	Token   string
	Prowl   string
	Forward string
}

const (
	Sforward = 1 << iota
	Sprowl
	Sspam
	Smaildir
)

var (
	p_service string
	bootstrap bool
	db        redis.Conn
)

func init() {
	flag.StringVar(&p_service, "add", "", "add service with random mail")
	flag.BoolVar(&bootstrap, "bootstrap", false, "import names & maillist, setup first user")
}

func import_names(filename, db_prefix string) {
	db, err := redis.Dial("tcp", ":6379")
	log.Printf("doing file: %s for db: %s\n", filename, db_prefix)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("could not open file: %s", filename)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	kvLine, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(kvLine)
		db.Do("sadd", db_prefix, s)
		kvLine, isPrefix, err = r.ReadLine()
	}
}

func import_primary_user() {
	db, _ := redis.Dial("tcp", ":6379")
	cur, _ := redis.Int(db.Do("get", "next.email.id"))
	for i := 0; i < cur; i++ {
		db.Do("sadd", "u0:ids", i)
	}

	db.Do("set", "u0:token", "")
	db.Do("set", "u0:prowl", "MY_PROWL_THING")
	db.Do("set", "u0:created", time.Now().Unix())
	db.Do("set", "u0:last", time.Now().Unix())
}

func import_maillist() {
	log.Println("importing maillist")
	f, _ := os.Open("/home/da/.maillist")
	defer f.Close()
	r := bufio.NewReader(f)

	service := make(map[string]string, 1)
	kvLine, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(kvLine)
		kv := strings.Split(s, "\t")
		service[kv[0]] = kv[1]
		kvLine, isPrefix, err = r.ReadLine()
	}

	for k, v := range service {
		c := Id{}
		if strings.Contains(k, "!") {
			c.State = 0
		} else {
			c.State = Sforward
		}

		c.Email = strings.Replace(k, "!", "@", -1)

		var user_id int
		user_id = 0

		//    user_str := strconv.Itoa(user_id)

		c.User = user_id
		c.Filter = ""
		c.Last = time.Now().Unix()
		c.Forwarded = 0
		c.Rejected = 0
		c.Service = v

		add_service(c.Email, c.Service, "", 0, c.State)
	}
}

func save() {
	db.Do("save")
}

func perform_bootstrap() {
	import_names("/home/da/.firstnames.txt", "names:given")
	import_names("/home/da/.lastnames.txt", "names:family")
	import_maillist()
	import_primary_user()
}

func generate_random_address() string {
	db, _ := redis.Dial("tcp", ":6379")
	name_g, _ := redis.String(db.Do("srandmember", "names:given"))
	name_f, _ := redis.String(db.Do("srandmember", "names:family"))

	rand_address := name_g + name_f
	rand.Seed(time.Now().Unix())
	chars := "abcdefghijklmnopqrstuvwxyz"
	runes := []rune(chars)
	for i := 0; i < 7; i++ {
		r := runes[rand.Intn(len(runes))]
		rand_address += string(r)
	}

	return strings.ToLower(rand_address + "@kreativlos.me")
}

func add_service(address, service_name, filter string, user, state int) {
	db, _ := redis.Dial("tcp", ":6379")

	current_email_id, _ := redis.Int(db.Do("get", "next.email.id"))
	u := "m" + strconv.Itoa(current_email_id) + ":"
	db.Do("set", u+"address", address)
	db.Do("set", u+"service", service_name)
	db.Do("set", u+"filter", filter)
	db.Do("set", u+"state", state)
	db.Do("set", u+"created", time.Now().Unix())
	db.Do("set", address, current_email_id)
	db.Do("set", u+"user", user)
	db.Do("sadd", "u"+strconv.Itoa(user)+":ids", current_email_id)
	db.Do("incr", "next.email.id")

	fmt.Printf("Service.: %s\n", service_name)
	fmt.Printf("E-mail..: %s\n", address)
	fmt.Printf("ID......: %d\n", current_email_id)
}

func add_random_service(service_name string, user int) {
	add_service(generate_random_address(), service_name, "", user, Sforward)
}

func main() {
	db, err := redis.Dial("tcp", ":6379")
	//	defer db.Close()

	if err != nil {
		log.Fatalf("db Connect failed: %s\n", err.Error())
	}

	flag.Parse()
	if !bootstrap {
		if p_service == "" {
			log.Fatal("need service name")
		} else {
			add_random_service(p_service, 0)
			rand.Seed(time.Now().Unix())
		}
	} else {
		perform_bootstrap()
	}

	db.Close()
}
