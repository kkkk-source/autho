package main

import (
	"encoding/json"
	"log"
	"net/http"
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

	token, err := createToken(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
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

func createToken(usrid uint64) (*TokenDetals, error) {
	td := &TokenDetals{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	td.AccessUUID = uuid.NewV4().String()
	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()
	td.RefreshUUID = uui.NewV4().String()

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

func init() {
	dsn := "localhost:6379"
	client := redis.NewClient(&redis.Options{
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
