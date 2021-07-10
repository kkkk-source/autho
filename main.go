package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/golang-jwt/jwt"
	"github.com/twinj/uuid"
)

type User struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var user = User{
	ID:       1,
	Username: "root",
	Password: "toor",
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
	}

	var u User
	d := json.NewDecoder(r.Body)
	if err := d.Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if user.Username != u.Username || user.Password != u.Password {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	td, err := createToken(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = createAuth(user.ID, td)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	token := map[string]string{
		"access_token":  td.AccessToken,
		"refresh_token": td.RefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(token)
}

type TokenDetails struct {
	AccessToken  string
	RefreshToken string
	AccessUUID   string
	RefreshUUID  string
	AtExpires    int64
	RtExpires    int64
}

func createToken(usrid uint64) (*TokenDetails, error) {
	var err error

	td := &TokenDetails{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	td.AccessUUID = uuid.NewV4().String()
	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()
	td.RefreshUUID = uuid.NewV4().String()

	// Create Access Token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUUID
	atClaims["user_id"] = usrid
	atClaims["exp"] = td.AtExpires

	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte("secret1"))
	if err != nil {
		return nil, err
	}

	// Create Refresh Token
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUUID
	rtClaims["user_id"] = usrid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte("secret2"))
	if err != nil {
		return nil, err
	}

	return td, nil
}

var client *redis.Client

func createAuth(usrid uint64, td *TokenDetails) error {
	at := time.Unix(td.AtExpires, 0) // convert unix to UTC
	rt := time.Unix(td.RtExpires, 0) // convert unix to UTC
	now := time.Now()

	err := client.Set(td.AccessUUID, strconv.Itoa(int(usrid)), at.Sub(now)).Err()
	if err != nil {
		return err
	}

	err = client.Set(td.RefreshUUID, strconv.Itoa(int(usrid)), rt.Sub(now)).Err()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	dsn := "db:6379"
	client = redis.NewClient(&redis.Options{
		Addr: dsn, // redis port
	})
	_, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
}

func main() {
	http.HandleFunc("/login", login)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
