package main

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/urfave/cli/v2"
	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var messagesCol *mongo.Collection
var usersCol *mongo.Collection
var ctx = context.TODO()

//type User struct {
//	ID	primitive.ObjectID `bson:"_id"`
//	Name string `bson:"text"`
//	Login string `bson:"text"`
//}
//
type Message struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Text      string             `bson:"text"`
	UserId    primitive.ObjectID `bson:"user_id"`
}

//
//type Friends struct {
//	ID        primitive.ObjectID `bson:"_id"`
//	User1Id   primitive.ObjectID `bson:"user1_id"`
//	User2Id   primitive.ObjectID `bson:"user2_id"`
//	StartedAt time.Time          `bson:"started_at"`
//}

type Friend struct {
	//ID        primitive.ObjectID `bson:"_id"`
	//CreatedAt time.Time `bson:"created_at"`
	//UpdatedAt time.Time `bson:"updated_at"`
	StartedAt time.Time          `bson:"started_at"`
	UserId    primitive.ObjectID `bson:"user_id"`
}

type User struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Name      string             `bson:"name"`
	Login     string             `bson:"login"`
	//Messages []*Message `bson:"messages"`
	Friends []*Friend `bson:"friends"`
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
				Name:    "addFriend",
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
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func createUser(user *User) error {
	_, err := usersCol.InsertOne(ctx, user)
	return err
}

func createMessage(message *Message) error {
	_, err := messagesCol.InsertOne(ctx, message)
	return err
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

	friendship := &Friend{
		StartedAt: time.Now(),
		UserId:    friend.ID,
	}

	if checkFriendShip(friend, user.ID) {
		return errors.New("Users are already friends")
	}

	filter := bson.D{primitive.E{Key: "login", Value: userLogin}}

	update := bson.D{primitive.E{Key: "$push", Value: bson.D{
		primitive.E{Key: "friends", Value: friendship},
	}}}

	u := &User{}
	return usersCol.FindOneAndUpdate(ctx, filter, update).Decode(u)
}

func countFriends(userLogin string) (int, error) {
	user, err := getUserByLogin(userLogin)
	if err != nil {
		return 0, err
	}
	return len(user.Friends), nil
}

func getFriendsNames(userLogin string) ([]string, error) {
	user, err := getUserByLogin(userLogin)
	if err != nil {
		return []string{}, err
	}
	var names []string
	for _, friendship := range user.Friends {
		friend, err := getUserByID(friendship.UserId)
		if err != nil {
			return []string{}, err
		}
		names = append(names, friend.Name)
	}
	return names, nil
}

func checkFriendShip(user *User, userIdForCheck primitive.ObjectID) bool {
	friends := filterFriendArray(user.Friends, userIdForCheck)
	if len(friends) == 0 {
		return false
	}
	return true
}

func filterFriendArray(friends []*Friend, userID primitive.ObjectID) []*Friend {
	var users []*Friend

	for _, friend := range friends {

		if friend.UserId == userID {
			users = append(users, friend)
		}
	}
	return users
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
	usersCol = client.Database("cheeper").Collection("users")
	messagesCol = client.Database("cheeper").Collection("messages")
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

	messagesArray := createTestMessages(numUsers, userIds)
	_, err = messagesCol.InsertMany(ctx, messagesArray)

	return err
}
func createTestUsers(numUsers int, userIds []primitive.ObjectID) []interface{} {
	usersArray := []interface{}{}
	var user *User
	var userId primitive.ObjectID
	var friendIndex int
	var friend *Friend
	timeNow := time.Now()
	createdAt, updatedAt := timeNow, timeNow
	for i := 0; i < numUsers; i++ {
		userId = userIds[i]

		friendIndex = randInt(0, numUsers)
		for friendIndex == i {
			friendIndex = randInt(0, numUsers)
		}

		friend = &Friend{
			StartedAt: createdAt,
			UserId:    userIds[friendIndex],
		}

		//numMessages = randInt(1, 6)
		//messages = make([]*Message, numMessages)
		//for j = 0; j < numMessages; j++ {
		//	message = &Message{
		//		//ID: primitive.NewObjectID(),
		//		CreatedAt: createdAt,
		//		UpdatedAt: updatedAt,
		//		Text: fmt.Sprintf("message number %d for user_%d", j, i),
		//		UserId: userId,
		//	}
		//	messages = append(messages, message)
		//}
		//j = 0

		user = &User{
			ID:        userId,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Name:      fmt.Sprintf("user_%d", i),
			Login:     fmt.Sprintf("login_%d", i),
			Friends:   []*Friend{friend},
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

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
