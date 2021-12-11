package test

import (
	"errors"
	"fmt"
	"github.com/DasHaSneg/tweeter/cheeper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func TestBdByNumReq(numReq int, isReading bool) (float64, error) {
	if isReading {
		return getTimeReading(numReq)
	}
	return getTimeInsertion(numReq)
}

func TestBdByArrayNumReq(numReqArray []int, isReading bool) ([]float64, error) {
	if len(numReqArray) != 0 {
		var tArray []float64
		for _, numReq := range numReqArray {
			t, err := TestBdByNumReq(numReq, isReading)
			if err != nil {
				return tArray, err
			}
			tArray = append(tArray, t)
		}
		return tArray, nil
	}
	return []float64{}, errors.New("The length of the array must be greater than zero")
}

func TestAllByArrayNumReq(numReqArray []int) ([]float64, []float64, error) {
	tArray1, err := TestBdByArrayNumReq(numReqArray, false)
	if err != nil {
		return nil, nil, err
	}
	tArray2, err := TestBdByArrayNumReq(numReqArray, true)
	if err != nil {
		return nil, nil, err
	}
	return tArray1, tArray2, nil
}

func getTimeInsertion(numReq int) (float64, error) {
	var messages []*cheeper.Message
	var i int
	user, err := cheeper.GetUserByLogin("login_0")
	if err != nil {
		return 0.0, err
	}
	for i = 0; i < numReq; i++ {
		messages = append(messages, cheeper.CreateMessage(user.ID, fmt.Sprintf("message number %d", i), time.Now()))
	}
	if len(messages) != 0 {
		start := time.Now()
		for i = 0; i < numReq; i++ {
			err = cheeper.SendMessage(messages[i])
		}
		duration := time.Since(start)
		return duration.Seconds(), nil
	} else {
		return 0.0, errors.New("Ошибка при создании сообщений")
	}
}

func getTimeReading(numReq int) (float64, error) {
	var i int
	id, err := primitive.ObjectIDFromHex("61b2a9a57df17e50f1958d21")
	if err != nil {
		return 0.0, err
	}
	_, err = cheeper.GetMessageByID(id)
	if err != nil {
		return 0.0, err
	}
	start := time.Now()
	for i = 0; i < numReq; i++ {
		cheeper.GetMessageByIdWithoutDecoding(id)
	}
	duration := time.Since(start)
	return duration.Seconds(), nil
}

func PrintAllTimes(tArray1 []float64, tArray2 []float64) {
	for i, t := range tArray1 {
		fmt.Printf("time writing %d: %f\n", i, t)
	}
	for i, t := range tArray2 {
		fmt.Printf("time reading %d: %f\n", i, t)
	}
}
