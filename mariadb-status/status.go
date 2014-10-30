package main

import (
	"bytes"
	_ "database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mariadb-tools/dbhelper"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var status map[string]int64
var prevStatus map[string]int64
var variable map[string]string

type timeSeries struct {
	Name    string
	Columns []string
	Points  [][]int64
}

var version = flag.Bool("version", false, "Return version")
var user = flag.String("user", "", "User for MariaDB login")
var password = flag.String("password", "", "Password for MariaDB login")
var host = flag.String("host", "", "MariaDB host IP address or FQDN")
var socket = flag.String("socket", "/var/run/mysqld/mysqld.sock", "Path of MariaDB unix socket")
var port = flag.String("port", "3306", "TCP Port of MariaDB server")

// Options specific to this command follow
var interval = flag.Int64("interval", 1, "Sleep interval for repeated commands")
var average = flag.Bool("average", false, "Average per second status data instead of aggregate")

func main() {

	flag.Parse()
	if *version == true {
		fmt.Println("MariaDB Tools version 0.0.1")
		os.Exit(0)
	}
	var address string
	if *socket != "" {
		address = "unix(" + *socket + ")"
	}
	if *host != "" {
		address = "tcp(" + *host + ":" + *port + ")"
	}

	// Create the database handle, confirm driver is present
	db, _ := sqlx.Open("mysql", *user+":"+*password+"@"+address+"/")
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	status = dbhelper.GetStatusAsInt(db)

	var iter uint64 = 0
	for {
		if (iter % 10) == 0 {
			fmt.Printf("  %-30s%-10s  %-10s  %-10s  %-10s  %-10s\n", "Queries", "Txns", "Threads", "Aborts", "Tables", "Files")
		}
		prevStatus = status
		status = dbhelper.GetStatusAsInt(db)
		fmt.Printf("%5s %5s %5s %5s %5s %5s %5s %5s %5s %5s %5s %5s %5s %5s %5s\n", "Que", "Sel", "Ins", "Upd", "Del", "Com", "Rbk", "Con", "Thr", "Cli", "Con", "Opn", "Opd", "Opn", "Opd")
		// fmt.Println("Com_select", status["COM_SELECT"])
		fmt.Printf("%5d %5d %5d %5d %5d %5d %5d %5d %5d %5d %5d %5d %5d %5d %5d\n", getCounter("QUERIES"), getCounter("COM_SELECT"), getCounter("COM_INSERT"), getCounter("COM_UPDATE"), getCounter("COM_DELETE"),
			getCounter("COM_COMMIT"), getCounter("COM_ROLLBACK"), getStatic("THREADS_CONNECTED"), getStatic("THREADS_RUNNING"), getCounter("ABORTED_CLIENTS"), getCounter("ABORTED_CONNECTS"),
			getStatic("OPEN_TABLES"), getCounter("OPENED_TABLES"), getStatic("OPEN_FILES"), getCounter("OPENED_FILES"))
		// storeStatus("Queries", getCounter("QUERIES"))
		time.Sleep(time.Duration(*interval) * time.Second)
		iter++
	}
}

func storeStatus(s string, u int64) {
	slice1 := []int64{u}
	slice2 := [][]int64{slice1}
	ts := timeSeries{s, []string{"value"}, slice2}
	// fmt.Printf("%s %d \n", ts.Name, ts.Points)
	b, err := json.Marshal(ts)
	js := string(b)
	js = "[" + js + "]"
	b = []byte(js)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBuffer(b)
	r, _ := http.Post("http://localhost:8086/db/mariadb/series?u=root&p=root", "text/json", body)
	response, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(response))
}

func getCounter(s string) int64 {
	if *average == true && *interval > 1 {
		return (status[s] - prevStatus[s]) / *interval
	} else {
		return status[s] - prevStatus[s]
	}
}

func getStatic(s string) int64 {
	return status[s]
}
