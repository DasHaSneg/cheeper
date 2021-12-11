package cheeper

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gookit/color.v1"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

var messageCol *mongo.Collection
var userCol *mongo.Collection
var friendshipCol *mongo.Collection
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
	UserId    primitive.ObjectID `bson:"user_id"`
	FriendId  primitive.ObjectID `bson:"friend_id"`
	StartedAt time.Time          `bson:"started_at"`
}

type User struct {
	ID        primitive.ObjectID `bson:"_id"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
	Name      string             `bson:"name"`
	Login     string             `bson:"login"`
}

func AddUser(name string, login string) error {
	timeNow := time.Now()
	user := &User{
		ID:        primitive.NewObjectID(),
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
		Name:      name,
		Login:     login,
	}

	return sendUser(user)
}

func sendUser(user *User) error {
	_, err := userCol.InsertOne(ctx, user)
	return err
}

func SendMessage(message *Message) error {
	_, err := messageCol.InsertOne(ctx, message)
	return err
}

func sendFriendship(friendship *Friendship) error {
	_, err := friendshipCol.InsertOne(ctx, friendship)
	return err
}

func CreateMessage(userID primitive.ObjectID, text string, timeNow time.Time) *Message {
	return &Message{
		ID:        primitive.NewObjectID(),
		CreatedAt: timeNow,
		UpdatedAt: timeNow,
		Text:      text,
		UserId:    userID,
	}
}

func AddMessage(userLogin string, text string) error {
	user, err := GetUserByLogin(userLogin)
	if err != nil {
		return err
	}
	timeNow := time.Now()
	message := CreateMessage(user.ID, text, timeNow)
	return SendMessage(message)
}

func AddFriend(userLogin string, friendLogin string) error {

	friend, err := GetUserByLogin(friendLogin)
	if err != nil {
		return err
	}

	user, err := GetUserByLogin(userLogin)
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
		UserId:    user.ID,
		FriendId:  friend.ID,
		StartedAt: time.Now(),
	}

	return sendFriendship(friendship)
}

func getArrayFromArrayString(arr []string) ([]int, error) {
	b := make([]int, len(arr))
	var err error
	for i, v := range arr {
		b[i], err = strconv.Atoi(v)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

func getTimeAndDateArrays(str string) ([]int, []int, error) {
	arr1 := strings.Split(str, " ")
	arr2 := strings.Split(arr1[0], ":")
	arr3 := strings.Split(arr1[1], "-")
	t, _ := getArrayFromArrayString(arr2)
	d, _ := getArrayFromArrayString(arr3)
	return t, d, nil
}

func GetMessagesByTime(userID primitive.ObjectID, startStr string, endStr string) ([]*Message, error) {
	t1, d1, _ := getTimeAndDateArrays(startStr)
	t2, d2, _ := getTimeAndDateArrays(endStr)
	start := time.Date(d1[2], time.Month(d1[1]), d1[0], t1[0], t1[1], 0, 0, time.UTC)
	end := time.Date(d2[2], time.Month(d2[1]), d2[0], t2[0], t2[1], 0, 0, time.UTC)
	filter := bson.M{"user_id": userID, "created_at": bson.M{"$gte": start, "$lte": end}}
	var messages []*Message
	cur, err := messageCol.Find(ctx, filter)
	if err != nil {
		return messages, err
	}

	for cur.Next(ctx) {
		var m Message
		err := cur.Decode(&m)
		if err != nil {
			return messages, err
		}
		messages = append(messages, &m)

	}

	if err := cur.Err(); err != nil {
		return messages, err
	}

	cur.Close(ctx)

	if len(messages) == 0 {
		return messages, mongo.ErrNoDocuments
	}

	return messages, nil
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

func CountFriends(userLogin string) (int32, error) {
	user, err := GetUserByLogin(userLogin)
	if err != nil {
		return 0, err
	}
	matchStage := bson.D{{"$match", bson.D{{"user_id", user.ID}}}}
	countStage := bson.D{{"$count", "total"}}

	showInfoCursor, err := friendshipCol.Aggregate(ctx, mongo.Pipeline{matchStage, countStage})
	if err != nil {
		panic(err)
	}
	var showsWithInfo []bson.M
	if err = showInfoCursor.All(ctx, &showsWithInfo); err != nil {
		panic(err)
	}
	return showsWithInfo[0]["total"].(int32), nil
}

func GetFriendsNames(userLogin string) ([]string, error) {
	user, err := GetUserByLogin(userLogin)
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

func PrintNames(names []string) {
	for _, n := range names {
		color.Green.Printf("%s\n", n)
	}
}

func PrintMessages(messages []*Message) {
	for _, m := range messages {
		color.Green.Printf("ID: %s ", m.ID)
		color.Green.Printf("CreatedAt: %s ", m.CreatedAt)
		color.Green.Printf("UpdatedAt: %s ", m.UpdatedAt)
		color.Green.Printf("UserId: %s ", m.UserId)
		color.Green.Printf("Text: %S\n", m.Text)
	}
}

func GetUserByLogin(login string) (*User, error) {
	filter := bson.D{primitive.E{Key: "login", Value: login}}
	u := &User{}
	err := userCol.FindOne(ctx, filter).Decode(u)
	return u, err
}

func getUserByID(userID primitive.ObjectID) (*User, error) {
	filter := bson.D{primitive.E{Key: "_id", Value: userID}}
	u := &User{}
	err := userCol.FindOne(ctx, filter).Decode(u)
	return u, err
}

func getFriendsIdsById(userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	filter := bson.D{{"user_id", userID}}
	var ids []primitive.ObjectID
	cur, err := friendshipCol.Find(ctx, filter)
	if err != nil {
		return ids, err
	}

	for cur.Next(ctx) {
		var f Friendship
		err := cur.Decode(&f)
		if err != nil {
			return ids, err
		}
		ids = append(ids, f.FriendId)

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

func GetMessageByID(id primitive.ObjectID) (*Message, error) {
	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	m := &Message{}
	err := messageCol.FindOne(ctx, filter).Decode(m)
	return m, err
}

func GetMessageByIdWithoutDecoding(id primitive.ObjectID) {
	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	messageCol.FindOne(ctx, filter)
}

func filterUsers(filter interface{}) ([]*User, error) {
	var users []*User

	cur, err := userCol.Find(ctx, filter)
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
	userCol = database.Collection("user")
	messageCol = database.Collection("message")
	friendshipCol = database.Collection("friendship")
}

func AddTestData(numUsers int) error {
	rand.Seed(time.Now().UTC().UnixNano())

	userIds := make([]primitive.ObjectID, numUsers)
	var i int
	for i = 0; i < 5; i++ {
		userIds[i] = primitive.NewObjectID()
	}

	usersArray := createTestUsers(numUsers, userIds)
	_, err := userCol.InsertMany(ctx, usersArray)

	friendshipsArray := createTestFriendships(numUsers, userIds)
	_, err = friendshipCol.InsertMany(ctx, friendshipsArray)

	messagesArray := createTestMessages(numUsers, userIds)
	_, err = messageCol.InsertMany(ctx, messagesArray)

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
			UserId:    userIds[i],
			FriendId:  userIds[friendIndex],
			StartedAt: timeNow,
		}
		friendshipsArray = append(friendshipsArray, friendship)
	}
	return friendshipsArray
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
