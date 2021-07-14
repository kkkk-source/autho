package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	Username: "abcd",
	Password: "dcba",
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
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

func logout(w http.ResponseWriter, r *http.Request) {
	au, err := extractTokenMetadata(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	deleted, err := deleteAuth(au.AccessUUID)
	if err != nil || deleted == 0 {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	msg := map[string]string{
		"message": "Successfully logged out",
		"status":  http.StatusText(http.StatusOK),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}

	tokenAuth, err := extractTokenMetadata(r)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	_, err = fetchAuth(tokenAuth)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	msg := map[string]string{
		"message": "Why are you here?",
		"status":  http.StatusText(http.StatusOK),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
}

func refresh(w http.ResponseWriter, r *http.Request) {
	mapToken := map[string]string{}
	d := json.NewDecoder(r.Body)
	if err := d.Decode(&mapToken); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	rt := mapToken["refresh_token"]
	token, err := jwt.Parse(rt, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v",
				token.Header["alg"])
		}

		return []byte("refresh_secret"), nil
	})

	// If there is an error, the token must have expired
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	// Is the token valid
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Since token is vaild, get the uuid
	// The token claims should conform to MapClaims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	refreshuuid, ok := claims["refresh_uuid"].(string)
	if !ok {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	usrID, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Delete the previous Refresh Token
	deleted, err := deleteAuth(refreshuuid)
	if err != nil || deleted == 0 {
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}

	// Create new pairs of refresh and access tokens
	ts, err := createToken(usrID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	}

	// Save the tokens metadata to redis
	err = createAuth(usrID, ts)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden),
			http.StatusForbidden)
		return
	}

	tokens := map[string]string{
		"access_token":  ts.AccessToken,
		"refresh_token": ts.RefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokens)
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
	td.AccessToken, err = at.SignedString([]byte("access_secret"))
	if err != nil {
		return nil, err
	}

	// Create Refresh Token
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUUID
	rtClaims["user_id"] = usrid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte("refresh_secret"))
	if err != nil {
		return nil, err
	}

	return td, nil
}

func deleteAuth(givenuuid string) (int64, error) {
	deleted, err := client.Del(givenuuid).Result()
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

type AccessDetails struct {
	AccessUUID string
	UserID     uint64
}

func fetchAuth(ad *AccessDetails) (uint64, error) {
	usrid, err := client.Get(ad.AccessUUID).Result()
	if err != nil {
		return 0, err
	}

	usrID, _ := strconv.ParseUint(usrid, 10, 64)
	return usrID, nil
}

func extractTokenMetadata(r *http.Request) (*AccessDetails, error) {
	token, err := verifyToken(r)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		accessUUID, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, err
		}

		usrid, err := strconv.ParseUint(fmt.Sprintf("%.f",
			claims["user_id"]), 10, 64)
		if err != nil {
			return nil, err
		}

		return &AccessDetails{
			AccessUUID: accessUUID,
			UserID:     usrid,
		}, nil
	}

	return nil, err
}

func tokenValid(r *http.Request) error {
	token, err := verifyToken(r)
	if err != nil {
		return err
	}

	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}

	return nil
}

func verifyToken(r *http.Request) (*jwt.Token, error) {
	tokenStr := extractToken(r)
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Make sure the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte("access_secret"), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

func extractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	str := strings.Split(bearToken, " ")
	if len(str) == 2 {
		return str[1]
	}

	return ""
}

var client *redis.Client

func createAuth(usrid uint64, td *TokenDetails) error {
	at := time.Unix(td.AtExpires, 0) // convert unix to UTC
	rt := time.Unix(td.RtExpires, 0) // convert unix to UTC
	now := time.Now()

	var err error

	err = client.Set(td.AccessUUID, strconv.Itoa(int(usrid)),
		at.Sub(now)).Err()
	if err != nil {
		return err
	}

	err = client.Set(td.RefreshUUID, strconv.Itoa(int(usrid)),
		rt.Sub(now)).Err()
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
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/refresh", refresh)
	http.HandleFunc("/say", sayHello)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
