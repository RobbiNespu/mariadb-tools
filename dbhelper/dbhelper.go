// dbhelper.go
package dbhelper

import (
	"database/sql"
	"log"
	"net"
	"strings"

	"github.com/jmoiron/sqlx"
)

const debug = false

type Processlist struct {
	Id       uint64
	User     string
	Host     string
	Database sql.NullString
	Command  string
	Time     float64
	State    string
}

type SlaveHosts struct {
	Server_id uint64
	Host      string
	Port      uint
	Master_id uint64
}

type MasterStatus struct {
	File             string
	Position         uint
	Binlog_Do_DB     string
	Binlog_Ignore_DB string
}

type SlaveStatus struct {
	Connection_name               string
	Slave_SQL_State               string
	Slave_IO_State                string
	Master_Host                   string
	Master_User                   string
	Master_Port                   uint
	Connect_Retry                 uint
	Master_Log_File               string
	Read_Master_Log_Pos           uint
	Relay_Log_File                string
	Relay_Log_Pos                 uint
	Relay_Master_Log_File         string
	Slave_IO_Running              string
	Slave_SQL_Running             string
	Replicate_Do_DB               string
	Replicate_Ignore_DB           string
	Replicate_Do_Table            string
	Replicate_Ignore_Table        string
	Replicate_Wild_Do_Table       string
	Replicate_Wild_Ignore_Table   string
	Last_Errno                    uint
	Last_Error                    string
	Skip_Counter                  uint
	Exec_Master_Log_Pos           uint
	Relay_Log_Space               uint
	Until_Condition               string
	Until_Log_File                string
	Until_Log_Pos                 uint
	Master_SSL_Allowed            string
	Master_SSL_CA_File            string
	Master_SSL_CA_Path            string
	Master_SSL_Cert               string
	Master_SSL_Cipher             string
	Master_SSL_Key                string
	Seconds_Behind_Master         sql.NullInt64
	Master_SSL_Verify_Server_Cert string
	Last_IO_Errno                 uint
	Last_IO_Error                 string
	Last_SQL_Errno                uint
	Last_SQL_Error                string
	Replicate_Ignore_Server_Ids   string
	Master_Server_Id              uint
	Master_SSL_Crl                string
	Master_SSL_Crlpath            string
	Using_Gtid                    string
	Gtid_IO_Pos                   string
	Retried_transactions          uint
	Max_relay_log_size            uint
	Executed_log_entries          uint
	Slave_received_heartbeats     uint
	Slave_heartbeat_period        float64
	Gtid_Slave_Pos                string
}

type Privileges struct {
	Select_priv      string
	Process_priv     string
	Super_priv       string
	Repl_slave_priv  string
	Repl_client_priv string
}

/* Connect to a MySQL server. Must be deprecated, use MySQLConnect instead */
func Connect(user string, password string, address string) *sqlx.DB {
	db, _ := sqlx.Open("mysql", user+":"+password+"@"+address+"/")
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func MySQLConnect(user string, password string, address string, parameters ...string) (*sqlx.DB, error) {
	dsn := user + ":" + password + "@" + address + "/"
	if len(parameters) > 0 {
		dsn += "?" + parameters[0]
	}
	db, err := sqlx.Connect("mysql", dsn)
	return db, err
}

func GetAddress(host string, port string, socket string) string {
	var address string
	if host != "" {
		address = "tcp(" + host + ":" + port + ")"
	} else {
		address = "unix(" + socket + ")"
	}
	return address
}

func GetProcesslist(db *sqlx.DB) []Processlist {
	pl := []Processlist{}
	err := db.Select(&pl, "SELECT id, user, host, `db` AS `database`, command, time_ms as time, state FROM INFORMATION_SCHEMA.PROCESSLIST")
	if err != nil {
		log.Fatalln("ERROR: Could not get processlist", err)
	}
	return pl
}

func GetPrivileges(db *sqlx.DB, user string, host string) (Privileges, error) {
	db.MapperFunc(strings.Title)
	priv := Privileges{}
	stmt := "SELECT Select_priv, Process_priv, Super_priv, Repl_slave_priv, Repl_client_priv FROM mysql.user WHERE user = ? AND host = ?"
	row := db.QueryRowx(stmt, user, host)
	err := row.StructScan(&priv)
	if err != nil {
		if err == sql.ErrNoRows {
			row := db.QueryRowx(stmt, user, "%")
			err = row.StructScan(&priv)
		}
		return priv, err
	}
	return priv, err
}

func GetSlaveStatus(db *sqlx.DB) (SlaveStatus, error) {
	db.MapperFunc(strings.Title)
	udb := db.Unsafe()
	ss := SlaveStatus{}
	err := udb.Get(&ss, "SHOW SLAVE STATUS")
	return ss, err
}

func GetMSlaveStatus(db *sqlx.DB, conn string) (SlaveStatus, error) {
	db.MapperFunc(strings.Title)
	udb := db.Unsafe()
	ss := SlaveStatus{}
	err := udb.Get(&ss, "SHOW SLAVE '"+conn+"' STATUS")
	return ss, err
}

func GetAllSlavesStatus(db *sqlx.DB) ([]SlaveStatus, error) {
	db.MapperFunc(strings.Title)
	udb := db.Unsafe()
	ss := []SlaveStatus{}
	err := udb.Select(&ss, "SHOW ALL SLAVES STATUS")
	return ss, err
}

func GetMasterStatus(db *sqlx.DB) (MasterStatus, error) {
	db.MapperFunc(strings.Title)
	ms := MasterStatus{}
	err := db.Get(&ms, "SHOW MASTER STATUS")
	return ms, err
}

func GetSlaveHosts(db *sqlx.DB) map[string]interface{} {
	rows, err := db.Queryx("SHOW SLAVE HOSTS")
	if err != nil {
		log.Fatalln("ERROR: Could not get slave hosts", err)
	}
	defer rows.Close()
	results := make(map[string]interface{})
	for rows.Next() {
		err = rows.MapScan(results)
		if err != nil {
			log.Fatal(err)
		}
	}
	return results
}

func GetSlaveHostsArray(db *sqlx.DB) []SlaveHosts {
	sh := []SlaveHosts{}
	err := db.Select(&sh, "SHOW SLAVE HOSTS")
	if err != nil {
		log.Fatalln("ERROR: Could not get slave hosts array", err)
	}
	return sh
}

func GetSlaveHostsDiscovery(db *sqlx.DB) []string {
	slaveList := []string{}
	/* This method does not return the server ports, so we cannot rely on it for the time being. */
	err := db.Select(&slaveList, "select host from information_schema.processlist where command ='binlog dump'")
	if err != nil {
		log.Fatalln("ERROR: Could not get slave hosts from the processlist", err)
	}
	return slaveList
}

func GetStatus(db *sqlx.DB) map[string]string {
	type Variable struct {
		Variable_name string
		Value         string
	}
	vars := make(map[string]string)
	rows, err := db.Queryx("SELECT Variable_name AS variable_name, Variable_Value AS value FROM information_schema.global_status")
	if err != nil {
		log.Fatalln("ERROR: Could not get status variable", err)
	}
	for rows.Next() {
		var v Variable
		err := rows.Scan(&v.Variable_name, &v.Value)
		if err != nil {
			log.Fatalln("ERROR: Could not get results from status scan", err)
		}
		vars[v.Variable_name] = v.Value
	}
	return vars
}

func GetStatusAsInt(db *sqlx.DB) map[string]int64 {
	type Variable struct {
		Variable_name string
		Value         int64
	}
	vars := make(map[string]int64)
	rows, err := db.Queryx("SELECT Variable_name AS variable_name, Variable_Value AS value FROM information_schema.global_status")
	if err != nil {
		log.Fatal("ERROR: Could not get status as integer", err)
	}
	for rows.Next() {
		var v Variable
		rows.Scan(&v.Variable_name, &v.Value)
		vars[v.Variable_name] = v.Value
	}
	return vars
}

func GetVariables(db *sqlx.DB) (map[string]string, error) {
	type Variable struct {
		Variable_name string
		Value         string
	}
	vars := make(map[string]string)
	rows, err := db.Queryx("SELECT Variable_name AS variable_name, Variable_Value AS value FROM information_schema.global_variables")
	if err != nil {
		return vars, err
	}
	for rows.Next() {
		var v Variable
		err := rows.Scan(&v.Variable_name, &v.Value)
		if err != nil {
			return vars, err
		}
		vars[v.Variable_name] = v.Value
	}
	return vars, err
}

func GetVariableByName(db *sqlx.DB, name string) string {
	var value string
	err := db.QueryRowx("SELECT Variable_Value AS Value FROM information_schema.global_variables WHERE Variable_Name = ?", name).Scan(&value)
	if err != nil {
		log.Println("ERROR: Could not get variable by name", err)
	}
	return value
}

func FlushTables(db *sqlx.DB) error {
	_, err := db.Exec("FLUSH TABLES")
	return err
}

func FlushTablesNoLog(db *sqlx.DB) error {
	_, err := db.Exec("FLUSH NO_WRITE_TO_BINLOG TABLES")
	return err
}

func FlushTablesWithReadLock(db *sqlx.DB) error {
	_, err := db.Exec("FLUSH TABLES WITH READ LOCK")
	return err
}

func UnlockTables(db *sqlx.DB) error {
	_, err := db.Exec("UNLOCK TABLES")
	return err
}

func StopSlave(db *sqlx.DB) error {
	_, err := db.Exec("STOP SLAVE")
	return err
}

func StartSlave(db *sqlx.DB) error {
	_, err := db.Exec("START SLAVE")
	return err
}

func ResetSlave(db *sqlx.DB, all bool) error {
	stmt := "RESET SLAVE"
	if all == true {
		stmt += " ALL"
	}
	_, err := db.Exec(stmt)
	return err
}

/* Check for a list of slave prerequisites.
- Slave is connected
- Binary log on
- Connected to master
- No replication filters
*/
func CheckSlavePrerequisites(db *sqlx.DB, s string) bool {
	if debug {
		log.Printf("CheckSlavePrerequisites called")
	}
	err := db.Ping()
	/* If slave is not online, skip to next iteration */
	if err != nil {
		log.Printf("WARN : Slave %s is offline. Skipping", s)
		return false
	}
	vars, _ := GetVariables(db)
	if vars["LOG_BIN"] == "OFF" {
		log.Printf("WARN : Binary log off. Slave %s cannot be used as candidate master.", s)
		return false
	}
	return true
}

func CheckBinlogFilters(m *sqlx.DB, s *sqlx.DB) bool {
	ms, err := GetMasterStatus(m)
	if err != nil {
		log.Println("ERROR: Can't check binlog status on master")
		return false
	}
	ss, err := GetMasterStatus(s)
	if err != nil {
		log.Println("ERROR: Can't check binlog status on slave")
		return false
	}
	if ms.Binlog_Do_DB == ss.Binlog_Do_DB && ms.Binlog_Ignore_DB == ss.Binlog_Ignore_DB {
		if debug {
			log.Println("INFO: Binlog filters check OK")
		}
		return true
	} else {
		if debug {
			log.Println("INFO: Binlog filters differ on both servers")
		}
		return false
	}
}

func CheckReplicationFilters(m *sqlx.DB, s *sqlx.DB) bool {
	mv, _ := GetVariables(m)
	sv, _ := GetVariables(s)
	if mv["REPLICATE_DO_TABLE"] == sv["REPLICATE_DO_TABLE"] && mv["REPLICATE_IGNORE_TABLE"] == sv["REPLICATE_IGNORE_TABLE"] && mv["REPLICATE_WILD_DO_TABLE"] == sv["REPLICATE_WILD_DO_TABLE"] && mv["REPLICATE_WILD_IGNORE_TABLE"] == sv["REPLICATE_WILD_IGNORE_TABLE"] && mv["REPLICATE_DO_DB"] == sv["REPLICATE_DO_DB"] && mv["REPLICATE_IGNORE_DB"] == sv["REPLICATE_IGNORE_DB"] {
		return true
	} else {
		return false
	}
}

/* Check if server is connected to declared master */
func IsSlaveof(db *sqlx.DB, s string, m string) bool {
	if debug {
		log.Printf("IsSlaveOf called")
	}
	ss, err := GetSlaveStatus(db)
	if err != nil {
		log.Printf("WARN : Server %s is not a slave. Skipping", s)
		return false
	}
	masterHost, err := CheckHostAddr(ss.Master_Host)
	if err != nil {
		log.Println("ERROR: Could not resolve master hostname", ss.Master_Host)
	}
	if masterHost != m {
		log.Printf("WARN : Slave %s is not connected to the current master %s (master_host=%s). Skipping", s, m, masterHost)
		return false
	}
	return true
}

/* Check if a slave is in sync with his master */
func CheckSlaveSync(dbS *sqlx.DB, dbM *sqlx.DB) bool {
	if debug {
		log.Printf("CheckSlaveSync called")
	}
	sGtid := GetVariableByName(dbS, "GTID_CURRENT_POS")
	mGtid := GetVariableByName(dbM, "GTID_CURRENT_POS")
	if sGtid == mGtid {
		return true
	} else {
		return false
	}
}

func MasterPosWait(db *sqlx.DB, gtid string) error {
	_, err := db.Exec("SELECT MASTER_GTID_WAIT(?)", gtid)
	return err
}

func SetReadOnly(db *sqlx.DB, flag bool) error {
	if flag == true {
		_, err := db.Exec("SET GLOBAL read_only=1")
		return err
	} else {
		_, err := db.Exec("SET GLOBAL read_only=0")
		return err
	}
}

func CheckLongRunningWrites(db *sqlx.DB, thresh int) int {
	var count int
	err := db.QueryRowx("select count(*) from information_schema.processlist where command = 'Query' and time >= ? and info not like 'select%'", thresh).Scan(&count)
	if err != nil {
		log.Println("ERROR: Could not check long running writes", err)
	}
	return count
}

func KillThreads(db *sqlx.DB) {
	var ids []int
	db.Select(&ids, "SELECT Id FROM information_schema.PROCESSLIST WHERE Command != 'binlog dump' AND User != 'system user' AND Id != CONNECTION_ID()")
	for _, id := range ids {
		db.Exec("KILL ?", id)
	}
}

/* Check if string is an IP address or a hostname, return a IP address */
func CheckHostAddr(h string) (string, error) {
	var err error
	if net.ParseIP(h) != nil {
		return h, err
	} else {
		ha, err := net.LookupHost(h)
		if err != nil {
			return "", err
		} else {
			return ha[0], err
		}
	}
}
