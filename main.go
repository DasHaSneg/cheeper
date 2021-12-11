package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/DasHaSneg/tweeter/cheeper"
	testBd "github.com/DasHaSneg/tweeter/test"
	"github.com/urfave/cli/v2"
	"gopkg.in/gookit/color.v1"
)

func main() {
	app := &cli.App{
		Name:  "cheeper",
		Usage: "A simple CLI program to manage your cheeper",
		Commands: []*cli.Command{
			{
				Name:    "addTestData",
				Aliases: []string{"d"},
				Usage:   "create and add test data",
				Action: func(c *cli.Context) error {
					str := c.Args().First()
					if str == "" {
						return errors.New("Specify the number of users")
					}

					numUsers, err := strconv.Atoi(str)
					if err != nil {
						return errors.New("The number of users must be a number")
					}
					return cheeper.AddTestData(numUsers)
				},
			},
			{
				Name:    "addUser",
				Aliases: []string{"u"},
				Usage:   "add user",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					login := args[0]
					name := args[1]
					if login == "" || name == "" {
						return errors.New("Specify user's login and name")
					}
					return cheeper.AddUser(name, login)
				},
			},
			{
				Name:    "addMessage",
				Aliases: []string{"m"},
				Usage:   "add message",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					login := args[0]
					text := args[1]
					if login == "" || text == "" {
						return errors.New("Specify user's login and text")
					}
					return cheeper.AddMessage(login, text)
				},
			},
			{
				Name:    "getMessagesLast24Hours",
				Aliases: []string{"m"},
				Usage:   "getMessages for the last 24 hours",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					login := args[0]
					start := args[1]
					end := args[2]
					if login == "" || start == "" || end == "" {
						return errors.New("Specify user's login, start date and end date. Example: hide \"16:00 10-12-2021\" \"16:00 11-12-2021\"")
					}
					user, err := cheeper.GetUserByLogin(login)
					if err != nil {
						return err
					}
					messages, err := cheeper.GetMessagesByTime(user.ID, start, end)
					if err != nil {
						return err
					}
					cheeper.PrintMessages(messages)
					return nil
				},
			},
			{
				Name:    "addFriendship",
				Aliases: []string{"f"},
				Usage:   "add friend",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					userLogin := args[0]
					friendLogin := args[1]
					if userLogin == "" || friendLogin == "" {
						return errors.New("Specify user login and friend login")
					}

					if userLogin == friendLogin {
						return errors.New("The username and friend's logins must be different.")
					}

					return cheeper.AddFriend(userLogin, friendLogin)
				},
			},
			{
				Name:    "getFriendsNames",
				Aliases: []string{"fn"},
				Usage:   "get friend's names",
				Action: func(c *cli.Context) error {
					login := c.Args().First()
					if login == "" {
						return errors.New("Specify user's login")
					}

					names, err := cheeper.GetFriendsNames(login)
					if err != nil {
						return err
					}
					cheeper.PrintNames(names)
					return nil
				},
			},
			{
				Name:    "countFriends",
				Aliases: []string{"cf"},
				Usage:   "get num of friend's",
				Action: func(c *cli.Context) error {
					login := c.Args().First()
					if login == "" {
						return errors.New("Specify user's login")
					}

					num, err := cheeper.CountFriends(login)
					if err != nil {
						return err
					}
					color.Green.Printf("%d", num)
					return nil
				},
			},
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "database test",
				Action: func(c *cli.Context) error {
					str := c.Args().First()
					if str == "" {
						return errors.New("Specify the number of requests as array. Example: [100, 1000]")
					}
					var numReqArray []int
					err := json.Unmarshal([]byte(str), &numReqArray)
					if err != nil {
						log.Fatal(err)
					}

					tArray1, tArray2, err := testBd.TestAllByArrayNumReq(numReqArray)
					if err != nil {
						return err
					}

					testBd.PrintAllTimes(tArray1, tArray2)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
