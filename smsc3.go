package main

import (
    "database/sql"
    "flag"
    "fmt"
    "os"
    "os/signal"
    "regexp"
    "sync"
    "time"

    "github.com/mdouchement/logger"
    "github.com/mdouchement/smsc3/smsc"
    "github.com/sirupsen/logrus"
)

func createTableAndInsert(tableName string, smppAddress, httpAddress, username, password string, max string, n int) error {
    db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/templates")
    if err != nil {
        return err
    }
    defer db.Close()

    createTableQuery := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s (
            SmppAddress VARCHAR(255),
            HttpAddress VARCHAR(255),
            Username VARCHAR(255),
            Password VARCHAR(255),
            Percent VARCHAR(255),
            Number_of_Binds INT
        )`, tableName)

    _, err = db.Exec(createTableQuery)
    if err != nil {
        return err
    }

    insertQuery := fmt.Sprintf("INSERT INTO %s (SmppAddress, HttpAddress, Username, Password, Percent, Number_of_Binds) VALUES (?, ?, ?, ?, ?, ?)", tableName)
    _, err = db.Exec(insertQuery, smppAddress, httpAddress, username, password, max, n)
    if err != nil {
        return err
    }

    return nil
}

func main() {
    smppAddrFlag := flag.String("smppaddr", ":20001", "SMPP server address")
    usernameFlag := flag.String("username", "hamza", "Username for SMSC")
    passwordFlag := flag.String("password", "12345678", "Password for SMSC")
    httpAddrFlag := flag.String("httpaddr", ":6000", "HTTP server address")
    per := flag.String("percent", "50", "Percent of Message that will be confirmed(DLRs)")
    n := flag.Int("n", 10, "Number of Binds per second")
    flag.Parse()

    l := logrus.New()
    l.SetFormatter(&logger.LogrusTextFormatter{
        DisableColors:   false,
        ForceColors:     true,
        ForceFormatting: true,
        PrefixRE:        regexp.MustCompile(`^(\[.*?\])\s`),
        FullTimestamp:   true,
        TimestampFormat: "2006-01-02 15:04:05",
    })

    tableName := "generator_REC"
    err := createTableAndInsert(tableName, *smppAddrFlag, *httpAddrFlag, *usernameFlag, *passwordFlag, *per, *n)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    fmt.Println("Table created and values inserted successfully.")

    s := &smsc.SMSC{
        SMPPaddr: *smppAddrFlag,
        Username: *usernameFlag,
        Password: *passwordFlag,
        HTTPaddr: *httpAddrFlag,
    }

    smsc.Initialize(logger.WrapLogrus(l), s)

    var wg sync.WaitGroup
    ticker := time.NewTicker(time.Second / time.Duration(*n)) // This creates a new tick every 1/n seconds
    defer ticker.Stop()

    go func() {
        for range ticker.C {
            if wg.Add(1); true {
                go func() {
                    defer wg.Done()
                    if err := s.Listen(*smppAddrFlag); err != nil {
                        l.Fatal(err)
                    }
                }()
            }
        }
    }()
    defer s.Stop()

    signals := make(chan os.Signal, 1)
    signal.Notify(signals, os.Interrupt, os.Kill)
    <-signals
    wg.Wait() // Ensure all goroutines finish before exiting
}
