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
				Usage:  "Start the clock",
				Action: handlePunchIn,
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name: "profile",
                        Aliases: []string{"p"},
                        Usage: "The profile to attach the session to",
                        Value: "default",
                    },
                },
			},
			{
				Name:   "out",
				Usage:  "Stop the clock",
				Action: handlePunchOut,
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name: "profile",
                        Aliases: []string{"p"},
                        Usage: "The profile the session is attached to",
                        Value: "default",
                    },
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
                Usage: "List sessions",
                Action: handlePunchList,
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name: "profile",
                        Aliases: []string{"p"},
                        Usage: "The profile to filter sessions by",
                        Value: "default",
                    },
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
            {
                Name: "profiles",
                Usage: "Manage punch profiles",
                Subcommands: []*cli.Command{
                    {
                        Name: "list",
                        Usage: "List all profiles",
                        Action: handlePunchProfilesList,
                    },
                    {
                        Name: "add",
                        Usage: "Create a new profile",
                        Action: handlePunchProfileAdd,
                        Flags: []cli.Flag{
                            &cli.StringFlag{
                                Name: "description",
                                Usage: "A human readable description of the profile",
                                Aliases: []string{"d"},
                                Value: "",
                            },
                        },
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

    profileSlug := c.String("profile")

	_, err = db.GetOpenSession(profileSlug)
	if err == nil {
		return errors.New("session already open, cannot punch in")
	} else if !errors.Is(err, ErrNoOpenSession) {
		return err
	}

	err = db.StartSession(profileSlug)
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

	currentSession, err := db.GetOpenSession(c.String("profile"))
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
        profileSlug: c.String("profile"),
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

func handlePunchProfilesList(c *cli.Context) error {
	db, err := getDB()
	if err != nil {
		return err
	}

    profiles, err := db.ListProfiles()
    if err != nil {
        return err
    }

    out, err := json.Marshal(profiles)
    if err != nil {
        return err
    }

    fmt.Println(string(out))
    
    return nil
}

func handlePunchProfileAdd(c *cli.Context) error {
	db, err := getDB()
	if err != nil {
		return err
	}

    var p Profile
    if c.Args().Len() != 1 {
        return fmt.Errorf("expected 1 argument, got %d", c.Args().Len())
    }
    p.Slug = c.Args().First()

    if d := c.String("description"); d != "" {
        p.Description = &d
    }

    err = db.CreateProfile(p)
    if err != nil {
        return err
    }

    fmt.Println("Profile added")

    return nil
}
