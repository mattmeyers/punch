package main

import (
	"encoding/json"
	"errors"
	"fmt"
    "time"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
    defaultSince := time.Time{}
    // This is a hardcoded date, it will never fail to parse.
    defaultBefore, _ := time.Parse(time.DateTime, "2099-01-01 00:00:00")

	app := &cli.App{
		Name:  "punch",
		Usage: "record the grind",
		Commands: []*cli.Command{
			{
				Name:   "in",
				Usage:  "start the clock",
				Action: handlePunchIn,
			},
			{
				Name:   "out",
				Usage:  "stop the clock",
				Action: handlePunchOut,
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name: "note",
                        Aliases: []string{"n"},
                        Usage: "Attach a note to the session",
                    },
                    &cli.BoolFlag{
                        Name: "delete",
                        Aliases: []string{"d"},
                        Usage: "Delete the current session",
                        Value: false,
                    },
                },
			},
            {
                Name: "list",
                Usage: "list sessions",
                Action: handlePunchList,
                Flags: []cli.Flag{
                    &cli.TimestampFlag{
                        Name: "since",
                        Aliases: []string{"s"},
                        Value: cli.NewTimestamp(defaultSince),
                        Layout: "2006-01-02",
                    },
                    &cli.TimestampFlag{
                        Name: "before",
                        Aliases: []string{"b"},
                        Value: cli.NewTimestamp(defaultBefore),
                        Layout: "2006-01-02",
                    },
                },
            },
		},
	}

	return app.Run(args)
}

func handlePunchIn(c *cli.Context) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	_, err = db.GetOpenSession()
	if err == nil {
		return errors.New("session already open, cannot punch in")
	} else if !errors.Is(err, ErrNoOpenSession) {
		return err
	}

	err = db.StartSession()
	if err != nil {
		return err
	}

	fmt.Println("Started session")

	return nil
}

func handlePunchOut(c *cli.Context) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	currentSession, err := db.GetOpenSession()
	if err != nil {
		return err
	}

    if c.Bool("delete") {
        return db.DeleteSession(currentSession.ID)
    }

    if note := c.String("note"); note != "" {
        currentSession.Note = &note
    }

	err = db.EndSession(currentSession)
	if err != nil {
		return err
	}

	fmt.Println("Session ended")
	return nil
}

func handlePunchList(c *cli.Context) error {
	db, err := getDB()
	if err != nil {
		return err
	}

    sessions, err := db.ListSessions(listFilters{
        since: *c.Timestamp("since"),
        before: *c.Timestamp("before"),
    })
    if err != nil {
        return err
    }

    out, err := json.Marshal(sessions)
    if err != nil {
        return err
    }

    fmt.Println(string(out))
    
    return nil
}
