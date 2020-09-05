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
	vkGroupID           = os.Getenv("GROUP_ID")       // Идентификатор группы
	vkGroupName         = os.Getenv("GROUP_NAME")     // Название группы

	vkWallID       = vkGroupName + "?w=wall-" + vkGroupID + "_"  // Идентификатор стены в формате xxx?w=wall-xxx_
	vkPhotoAlbumID = "photo-" + vkGroupID + "_"                  // Идентификатор фотоальбома в формате photo-xxx_
	vkVideoID      = vkGroupName + "?z=video-" + vkGroupID + "_" // Идентификатор видео
	vkPhotoID      = vkGroupName + "?z=photo-" + vkGroupID + "_" // Идентификатор фото
	vkTopicID      = "https://vk.com/topic-" + vkGroupID + "_"
)

type vkEvents struct {
	Type   string `json:"type"`
	Object struct {
		UserID     int      `json:"user_id"`  // для фото
		FromID     int      `json:"from_id"`  // для комментариев
		OwnerID    int      `json:"owner_id"` // для аудио и видео
		LikerID    int      `json:"liker_id"` // для лайков
		ID         int      `json:"id"`       // идентификатор фото или комментария
		PhotoID    int      `json:"photo_id"`
		PostID     int      `json:"post_id"`
		TopicID    int      `json:"topic_id"`
		ItemID     int      `json:"item_id"`
		PollID     int      `json:"poll_id"`
		Title      string   `json:"title"`       // название композиции.
		ObjectType string   `json:"object_type"` // для лайков
		ObjectID   int      `json:"object_id"`
		JoinType   string   `json:"join_type"`
		AlbumID    int      `json:"album_id"` // идентификатор альбома, в котором находится фотография
		Text       string   `json:"text"`     // текст описания
		Message    struct { // Личное сообщение
			ID     int    `json:"id"`      // идентификатор сообщения
			Date   int    `json:"date"`    // время отправки в Unixtime
			FromID int    `json:"from_id"` // идентификатор отправителя
			Text   string `json:"text"`    // текст сообщения
		} `json:"message"`
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

	case "message_reply":
		// новое исходящее сообщение, возникает каждый раз при отправке сообщения и зацикливается, если по факту этого события происходит снова отправка сообщения
		return "ok", nil

	case "message_typing_state":
		// кто-то набирает сообщение, может быть очень много уведомлений
		return "ok", nil

	// Раздел Сообщения
	case "message_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.Message.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "входящее сообщение от " + lastName + " " + firstName + " https://vk.com/id" + userID + ": " + event.Object.Message.Text
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "message_allow":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.Message.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "подписка на сообщения от сообщества:" + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "message_deny":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.Message.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "новый запрет сообщений от сообщества:" + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Фотографии

	case "photo_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.Message.FromID)
		photoID := strconv.Itoa(event.Object.ID)
		firstName, lastName := getUserInfo(userID)

		message := "добавление фотографии в альбом" + vkPhotoAlbumID + strconv.Itoa(event.Object.AlbumID) + " от " + lastName + " " + firstName + " https://vk.com/id" + userID + " фото " + vkPhotoAlbumID + photoID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "photo_comment_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.Message.FromID)
		photoID := strconv.Itoa(event.Object.PhotoID)
		firstName, lastName := getUserInfo(userID)

		message := "Добавлен комментарий под фото https://vk.com/" + vkPhotoAlbumID + photoID + " " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "photo_comment_edit":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.Message.FromID)
		photoID := strconv.Itoa(event.Object.PhotoID)
		firstName, lastName := getUserInfo(userID)

		message := "Отредактирован комментарий под фото https://vk.com/" + vkPhotoAlbumID + photoID + " " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "photo_comment_delete":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		photoID := strconv.Itoa(event.Object.PhotoID)
		firstName, lastName := getUserInfo(userID)

		message := "Удален комментарий под фото https://vk.com/" + vkPhotoAlbumID + photoID + " " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Аудиозаписи
	case "audio_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.OwnerID)
		title := event.Object.Title
		firstName, lastName := getUserInfo(userID)

		message := "Добавлена аудиозапись " + title + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Видеозаписи
	case "video_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.OwnerID)
		title := event.Object.Title
		firstName, lastName := getUserInfo(userID)

		message := "Добавлена видеозапись " + title + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Записи на стене
	case "wall_post_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "Добавлена запись на стене: " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "wall_repost":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "Добавлен репост записи на стене: " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "wall_reply_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		postID := strconv.Itoa(event.Object.PostID)
		firstName, lastName := getUserInfo(userID)

		message := lastName + " " + firstName + " https://vk.com/id" + userID + " оставил комментарий на стене: " + event.Object.Text + " ссылка на запись https://vk.com/" + vkWallID + postID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Отметки "Мне нравится"
	case "like_add":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.LikerID)
		var object string
		switch event.Object.ObjectType {
		case "post":
			object = "под записью " + vkWallID + strconv.Itoa(event.Object.ObjectID)
		case "video":
			object = "под видеозаписью " + vkVideoID + strconv.Itoa(event.Object.ObjectID)
		case "photo":
			object = "под фото " + vkPhotoID + strconv.Itoa(event.Object.ObjectID)
		case "comment":
			object = "под комментарием в записи " + vkWallID+ strconv.Itoa(event.Object.ObjectID)
		case "note":
			object = "под заметкой " + strconv.Itoa(event.Object.ObjectID)
		case "topic_comment":
			object = "под комментарием в обсуждении " + strconv.Itoa(event.Object.ObjectID)
		case "photo_comment":
			object = "под комментарием к фото " + strconv.Itoa(event.Object.ObjectID)
		case "video_comment":
			object = "под комментарием к видео " + strconv.Itoa(event.Object.ObjectID)
		case "market":
			object = "под товаром " + strconv.Itoa(event.Object.ObjectID)
		case "market_comment":
			object = "под комментарием к товару " + strconv.Itoa(event.Object.ObjectID)
		default:
			object = "под " + event.Object.ObjectType + strconv.Itoa(event.Object.ObjectID)
		}

		firstName, lastName := getUserInfo(userID)

		message := lastName + " " + firstName + " https://vk.com/id" + userID + " поставил лайк " + object
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "like_remove":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.LikerID)
		var object string
		switch event.Object.ObjectType {
		case "post":
			object = "под записью " + vkWallID + strconv.Itoa(event.Object.ObjectID)
		case "video":
			object = "под видеозаписью " + strconv.Itoa(event.Object.ObjectID)
		case "photo":
			object = "под фото " + vkPhotoAlbumID + strconv.Itoa(event.Object.ObjectID)
		case "comment":
			object = "под комментарием " + strconv.Itoa(event.Object.ObjectID)
		case "note":
			object = "под заметкой " + strconv.Itoa(event.Object.ObjectID)
		case "topic_comment":
			object = "под комментарием в обсуждении " + strconv.Itoa(event.Object.ObjectID)
		case "photo_comment":
			object = "под комментарием к фото " + strconv.Itoa(event.Object.ObjectID)
		case "video_comment":
			object = "под комментарием к видео " + strconv.Itoa(event.Object.ObjectID)
		case "market":
			object = "под товаром " + strconv.Itoa(event.Object.ObjectID)
		case "market_comment":
			object = "под комментарием к товару " + strconv.Itoa(event.Object.ObjectID)
		default:
			object = "под " + event.Object.ObjectType + strconv.Itoa(event.Object.ObjectID)
		}

		firstName, lastName := getUserInfo(userID)

		message := lastName + " " + firstName + " https://vk.com/id" + userID + " удалил лайк " + object

		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Обсуждения
	case "board_post_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "Создан комментарий в обсуждении: " + vkTopicID + strconv.Itoa(event.Object.TopicID) + " с текстом" + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "board_post_edit":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "Отредактирован комментарий в обсуждении: " + vkTopicID + strconv.Itoa(event.Object.TopicID) + " с текстом" + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "board_post_delete":
		// message := event.Object.JoinType

		message := "Удален комментарий в обсуждении: " + vkTopicID + strconv.Itoa(event.Object.TopicID)
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Товары
	case "market_comment_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "Новый комментарий к товару: " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID + "идентификатор товара " + strconv.Itoa(event.Object.ItemID)
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "market_comment_edit":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.FromID)
		firstName, lastName := getUserInfo(userID)

		message := "Редактирование комментария к товару: " + event.Object.Text + " от " + lastName + " " + firstName + " https://vk.com/id" + userID + "идентификатор товара " + strconv.Itoa(event.Object.ItemID)
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

	case "market_comment_delete":
		// message := event.Object.JoinType

		message := "Удаление комментария к товару: " + "идентификатор товара " + strconv.Itoa(event.Object.ItemID)
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Пользователи
	case "group_leave":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.UserID)
		firstName, lastName := getUserInfo(userID)

		message := lastName + " " + firstName + " https://vk.com/id" + userID + " покинул группу"
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
		message := lastName + " " + firstName + " https://vk.com/id" + userID + " вступил в группу" + joinMessage

		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
		return "ok", nil

		// Раздел Прочее
	case "poll_vote_new":
		// message := event.Object.JoinType
		userID := strconv.Itoa(event.Object.UserID)
		firstName, lastName := getUserInfo(userID)

		message := "добавление голоса в публичном опросе: " + strconv.Itoa(event.Object.PollID) + " от " + lastName + " " + firstName + " https://vk.com/id" + userID + "идентификатор товара " + strconv.Itoa(event.Object.ItemID)
		sendMessage(message, sendToUserID)
		sendMessage(message, sendToUserIDControl)
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
	q.Add("random_id", "0")

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
