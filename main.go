package main

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/urfave/cli/v2"
	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gookit/color.v1"
)

var messagesCol *mongo.Collection
var usersCol *mongo.Collection
var friendshipsCol *mongo.Collection
var ctx = context.TODO()

type Message struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Text      string             `bson:"text"`
	UserId    primitive.ObjectID `bson:"user_id"`
}

type Friendship struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	User1Id   primitive.ObjectID `bson:"user1_id"`
	User2Id   primitive.ObjectID `bson:"user2_id"`
	StartedAt time.Time          `bson:"started_at"`
}

type User struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Name      string             `bson:"name"`
	Login     string             `bson:"login"`
	//Messages []*Message `bson:"messages"`
	//Friends []*Friend `bson:"friends"`
}

func main() {
	app := &cli.App{
		Name:  "cheeper",
		Usage: "A simple CLI program to manage your cheeper",
		Commands: []*cli.Command{
			{
				Name:    "addTestData",
				Aliases: []string{"t"},
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
					return addTestData(numUsers)
				},
			},
			{
				Name:    "createUser",
				Aliases: []string{"u"},
				Usage:   "add user",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					login := args[0]
					name := args[1]
					if login == "" || name == "" {
						return errors.New("Specify user's login and name")
					}
					timeNow := time.Now()
					user := &User{
						ID:        primitive.NewObjectID(),
						CreatedAt: timeNow,
						UpdatedAt: timeNow,
						Name:      name,
						Login:     login,
					}

					return sendUser(user)
				},
			},
			{
				Name:    "createMessage",
				Aliases: []string{"m"},
				Usage:   "add message",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					login := args[0]
					text := args[1]
					if login == "" || text == "" {
						return errors.New("Specify user's login and text")
					}
					return addMessage(login, text)
				},
			},
			{
				Name:    "createFriendship",
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

					return addFriend(userLogin, friendLogin)
				},
			},
			{
				Name:    "getFriendName",
				Aliases: []string{"fn"},
				Usage:   "get friend's names",
				Action: func(c *cli.Context) error {
					login := c.Args().First()
					if login == "" {
						return errors.New("Specify user's login")
					}

					names, err := getFriendsNames(login)
					if err != nil {
						return err
					}
					printNames(names)
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

					num, err := countFriends(login)
					if err != nil {
						return err
					}
					color.Green.Printf("%d", num)
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

func sendUser(user *User) error {
	_, err := usersCol.InsertOne(ctx, user)
	return err
}

func sendMessage(message *Message) error {
	_, err := messagesCol.InsertOne(ctx, message)
	return err
}

func sendFriendship(friendship *Friendship) error {
	_, err := friendshipsCol.InsertOne(ctx, friendship)
	return err
}

func addMessage(userLogin string, text string) error {
	user, err := getUserByLogin(userLogin)
	if err != nil {
		return err
	}
	timeNow := time.Now()
	message := &Message{
		ID:        primitive.NewObjectID(),
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
		Text:      text,
		UserId:    user.ID,
	}
	return sendMessage(message)
}

func addFriend(userLogin string, friendLogin string) error {

	friend, err := getUserByLogin(friendLogin)
	if err != nil {
		return err
	}

	user, err := getUserByLogin(userLogin)
	if err != nil {
		return err
	}

	if checkFriendShip(user.ID, friend.ID) {
		return errors.New("Users are already friends")
	}

	timeNow := time.Now()
	friendship := &Friendship{
		ID:        primitive.NewObjectID(),
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
		User1Id:   user.ID,
		User2Id:   user.ID,
		StartedAt: time.Now(),
	}

	return sendFriendship(friendship)
}

func checkFriendShip(userId primitive.ObjectID, friendId primitive.ObjectID) bool {
	ids, err := getFriendsIdsById(userId)
	if err != nil {
		return false
	}
	for _, i := range ids {
		if i == friendId {
			return true
		}
	}
	return false
}

func countFriends(userLogin string) (int32, error) {
	user, err := getUserByLogin(userLogin)
	if err != nil {
		return 0, err
	}
	matchStage := bson.D{{"$match", bson.D{{"user1_id", user.ID}}}}
	countStage := bson.D{{"$count", "total"}}

	showInfoCursor, err := friendshipsCol.Aggregate(ctx, mongo.Pipeline{matchStage, countStage})
	if err != nil {
		panic(err)
	}
	var showsWithInfo []bson.M
	if err = showInfoCursor.All(ctx, &showsWithInfo); err != nil {
		panic(err)
	}
	return showsWithInfo[0]["total"].(int32), nil
}

func getFriendsNames(userLogin string) ([]string, error) {
	user, err := getUserByLogin(userLogin)
	if err != nil {
		return []string{}, err
	}

	friendsIds, err := getFriendsIdsById(user.ID)
	if err != nil {
		return []string{}, err
	}
	var names []string
	for _, friendId := range friendsIds {
		friend, err := getUserByID(friendId)
		if err != nil {
			return []string{}, err
		}
		names = append(names, friend.Name)
	}
	sort.Strings(names)
	return names, nil
}

func printNames(names []string) {
	for _, n := range names {
		color.Green.Printf("%s\n", n)
	}
}

func getUserByLogin(login string) (*User, error) {
	filter := bson.D{primitive.E{Key: "login", Value: login}}
	u := &User{}
	err := usersCol.FindOne(ctx, filter).Decode(u)
	return u, err
}

func getUserByID(userID primitive.ObjectID) (*User, error) {
	filter := bson.D{primitive.E{Key: "_id", Value: userID}}
	u := &User{}
	err := usersCol.FindOne(ctx, filter).Decode(u)
	return u, err
}

func getFriendsIdsById(userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	filter := bson.D{primitive.E{Key: "user1_id", Value: userID}}
	var ids []primitive.ObjectID
	cur, err := friendshipsCol.Find(ctx, filter)
	if err != nil {
		return ids, err
	}

	for cur.Next(ctx) {
		var f Friendship
		err := cur.Decode(&f)
		if err != nil {
			return ids, err
		}
		ids = append(ids, f.User2Id)
	}

	if err := cur.Err(); err != nil {
		return ids, err
	}

	cur.Close(ctx)

	if len(ids) == 0 {
		return ids, mongo.ErrNoDocuments
	}

	return ids, nil
}

func filterUsers(filter interface{}) ([]*User, error) {
	var users []*User

	cur, err := usersCol.Find(ctx, filter)
	if err != nil {
		return users, err
	}

	for cur.Next(ctx) {
		var t User
		err := cur.Decode(&t)
		if err != nil {
			return users, err
		}
		users = append(users, &t)
	}

	if err := cur.Err(); err != nil {
		return users, err
	}

	cur.Close(ctx)

	if len(users) == 0 {
		return users, mongo.ErrNoDocuments
	}

	return users, nil
}

func init() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	database := client.Database("cheeper")
	usersCol = database.Collection("users")
	messagesCol = database.Collection("messages")
	friendshipsCol = database.Collection("friendships")
}

func addTestData(numUsers int) error {
	rand.Seed(time.Now().UTC().UnixNano())

	userIds := make([]primitive.ObjectID, numUsers)
	var i int
	for i = 0; i < 5; i++ {
		userIds[i] = primitive.NewObjectID()
	}

	usersArray := createTestUsers(numUsers, userIds)
	_, err := usersCol.InsertMany(ctx, usersArray)

	friendshipsArray := createTestFriendships(numUsers, userIds)
	_, err = friendshipsCol.InsertMany(ctx, friendshipsArray)

	messagesArray := createTestMessages(numUsers, userIds)
	_, err = messagesCol.InsertMany(ctx, messagesArray)

	return err
}
func createTestUsers(numUsers int, userIds []primitive.ObjectID) []interface{} {
	usersArray := []interface{}{}
	var user *User
	var userId primitive.ObjectID
	timeNow := time.Now()
	createdAt, updatedAt := timeNow, timeNow
	for i := 0; i < numUsers; i++ {
		userId = userIds[i]
		user = &User{
			ID:        userId,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Name:      fmt.Sprintf("user_%d", i),
			Login:     fmt.Sprintf("login_%d", i),
		}

		usersArray = append(usersArray, user)
	}
	return usersArray
}

func createTestMessages(numUsers int, userIds []primitive.ObjectID) []interface{} {
	messagesArray := []interface{}{}
	var userIndex int
	var message *Message
	timeNow := time.Now()
	createdAt, updatedAt := timeNow, timeNow
	for i := 0; i < 10; i++ {
		userIndex = randInt(0, numUsers)
		message = &Message{
			ID:        primitive.NewObjectID(),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Text:      fmt.Sprintf("message number %d for user_%d", i, userIndex),
			UserId:    userIds[userIndex],
		}
		messagesArray = append(messagesArray, message)
	}
	return messagesArray
}

func createTestFriendships(numUsers int, userIds []primitive.ObjectID) []interface{} {
	friendshipsArray := []interface{}{}
	var friendIndex int
	var friendship *Friendship
	timeNow := time.Now()
	for i := 0; i < numUsers; i++ {
		friendIndex = randInt(0, numUsers)
		for friendIndex == i {
			friendIndex = randInt(0, numUsers)
		}
		friendship = &Friendship{
			ID:        primitive.NewObjectID(),
			CreatedAt: timeNow,
			UpdatedAt: timeNow,
			User1Id:   userIds[i],
			User2Id:   userIds[friendIndex],
			StartedAt: timeNow,
		}
		friendshipsArray = append(friendshipsArray, friendship)
	}
	return friendshipsArray
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
