package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/namsral/flag"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()

	var bindAddr string
	var bindPort int
	var keyVal string
	var getOnly bool
	flags := flag.NewFlagSet("GoRedis", flag.ContinueOnError)
	flags.StringVar(&bindAddr, "bind_addr", "", "bind_addr=<IP>")
	flags.IntVar(&bindPort, "bind_port", 8090, "bind_port=<INT>")

	flags.StringVar(&keyVal, "key_val", "", "key_value=key:val to set. empty string is allowed")
	flags.BoolVar(&getOnly, "get_only", false, "get_only=true in combination with key_val will only get a value for you")

	flags.SetOutput(io.Discard)
	err := flags.Parse(os.Args[1:])

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Printf("GoRedis\n\n")
			fmt.Printf("USAGE\n\n  %s [OPTIONS]\n\n", os.Args[0])
			fmt.Print("OPTIONS\n\n")
			flags.SetOutput(os.Stdout)
			flags.PrintDefaults()
			os.Exit(0)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	conn, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", bindAddr, bindPort))
	if err != nil {
		log.Fatal(err)
	}
	// Importantly, use defer to ensure the connection is always
	// properly closed before exiting the main() function.
	defer func() { _ = conn.Close() }()

	key := keyVal

	if !getOnly {

		key = "key"
		val := "mu_super_long_value"

		// 3 symbols is minimum: a:b is correct
		if keyVal != "" && len(keyVal) >= 3 {
			if i := strings.IndexByte(keyVal, ':'); i > 0 && i < len(keyVal)-1 {
				key = keyVal[:i]
				val = keyVal[i+1:]
			}
		}

		log.Printf("SET: %s:%s\n", key, val)
		_, err = conn.Do("SET", key, val)
		if err != nil {
			log.Fatal(err)
		}
	}

	reply, err := conn.Do("GET", key)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("RESPONSE: %v\n", reply)
}
