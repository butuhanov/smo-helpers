package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	confirmationToken   = os.Getenv("CONFIRMATION_TOKEN")
	token               = os.Getenv("TOKEN")
	errorBackend        = errors.New("\"Something went wrong\"")
	myClient            = &http.Client{Timeout: 60 * time.Second}
	vkAPIversion        = os.Getenv("VKAPI")          // Версия API
	sendToUserID        = os.Getenv("USERID")         // Пользователь, которому будут отправляться уведомления
	sendToUserIDControl = os.Getenv("USERID_CONTROL") // Дополнительно отправлять сообщения пользователю
	vkPhotoAlbumID      = os.Getenv("PHOTO_ALBUM_ID") // Идентификатор фотоальбома в формате photo-xxx_
	vkWallID            = os.Getenv("WALL_ID")        // Идентификатор стены в формате xxx?w=wall-xxx_

)

type vkEvents struct {
	Type   string `json:"type"`
	Object struct {
		ID int `json:"id"` // идентификатор сообщения
		Date int  `json:"date"` // время отправки в Unixtime
	//	UserID   int    `json:"user_id"` // устарело с версии 5.80
		FromID   int    `json:"from_id"` // идентификатор отправителя
		PhotoID  int    `json:"photo_id"`
		PostID   int    `json:"post_id"`
		JoinType string `json:"join_type"`
		Text     string `json:"text"`
	} `json:"object"`
	GroupID int `json:"group_id"`
}

type user struct {
	Response []struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"response"`
}
type response struct {
	Response int `json:"response"`
}

func handleLambdaEvent(event vkEvents) (string, error) {

	switch event.Type {

		// Тестовые и системные сообщения
	case "confirmation":
		return confirmationToken, nil

	case "test_connection":
		message := "проверка связи"

		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)

		return "ok", nil

	case "message_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "входящее сообщение:" + event.Object.Text + " от пользователя " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil


	case "message_reply":
		// новое исходящее сообщение, возникает каждый раз при отправке сообщения и зацикливается, если по факту этого события происходит снова отправка сообщения
		return "ok", nil

	case "group_leave":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.UserID)
		firstName, lastName := getUserInfo(userID)

		message := "Пользователь " + lastName + " " + firstName + " https://vk.com/id" + userID + " покинул группу"
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "photo_comment_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		photoID := strconv.Itoa(event.Object.PhotoID)
		firstName, lastName := getUserInfo(userID)

		message := "Добавлен комментарий под фото https://vk.com/" + vkPhotoAlbumID + photoID + " " + event.Object.Text + " от пользователя " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "photo_comment_edit":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		photoID := strconv.Itoa(event.Object.PhotoID)
		firstName, lastName := getUserInfo(userID)

		message := "Отредактирован комментарий под фото https://vk.com/" + vkPhotoAlbumID + photoID + " " + event.Object.Text + " от пользователя " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "wall_reply_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		postID := strconv.Itoa(event.Object.PostID)
		firstName, lastName := getUserInfo(userID)

		message := "Пользователь " + lastName + " " + firstName + " https://vk.com/id" + userID + " оставил комментарий на стене: " + event.Object.Text + " ссылка на запись https://vk.com/" + vkWallID + postID + "%2Fall"
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil



	case "group_join":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.UserID)
		firstName, lastName := getUserInfo(userID)
		joinType := event.Object.JoinType
		var joinMessage string
		switch joinType {
		case "accepted":
			joinMessage = "принял приглашение"
		case "request":
			joinMessage = "подал приглашение"
		}
		message := "Пользователь " + lastName + " " + firstName + " https://vk.com/id" + userID + " вступил в группу" + joinMessage

		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil


	case "message_typing_state":
		// кто-то набирает сообщение
		return "ok", nil

	default:
		message := "Произошло событие:" + event.Type
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil
	}

	// return "\"error\"", errorBackend
}

// sendMessage отправляет сообщение пользователю
func sendMessage(message, userID string) {

	url := "https://api.vk.com/method/messages.send"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	checkErr(err, "sendMessage:http.NewRequest")

	q := req.URL.Query()
	q.Add("message", message)
	q.Add("user_id", userID)
	q.Add("access_token", token)
	q.Add("v", vkAPIversion)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	checkErr(err, "sendMessage:client.Do")

	echo, err := ioutil.ReadAll(resp.Body)
	checkErr(err, "sendMessage:ioutil.ReadAll")
	fmt.Printf("%s\r\n", echo)
	defer resp.Body.Close()

}

// getUserInfo получает информацию о пользователе
func getUserInfo(userID string) (string, string) {

	vkURL := "https://api.vk.com/method/users.get?user_ids=" + userID + "&access_token=" + token + "&v=" + vkAPIversion
	user := new(user) // or &User{}
	getJSON(vkURL, user)
	return user.Response[0].FirstName, user.Response[0].LastName

	// slcB, _ := json.Marshal(event)
	// fmt.Println(string(slcB))
}

func keepLines(s string, n int) string {
	result := strings.Join(strings.Split(s, "\n")[:n], "\n")
	return strings.Replace(result, "\r", "", -1)
}

func checkErr(err error, message string) {
	if err != nil {
		log.Print("error:" + message)
		log.Print(err.Error())
	}

}

func getJSON(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func main() {
	lambda.Start(handleLambdaEvent)
}
