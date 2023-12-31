package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"k8s.io/api/imagepolicy/v1alpha1"
	"os"
	"go.uber.org/zap"
	"strings"
	"strconv"
)

var logger = zap.Must(zap.NewDevelopment())
var WHITELIST = []string{}
var EXCLUDE_NAMESPACES = []string{}
var DOCKER_HOST = "unix:///var/run/docker.sock"
var REPLICATOR_ENV_VARS = make(map[string]string)
var REPLICATOR_ON = false
var SCANNER_ON = false

func imagePolicyWebhookHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != "/ipw" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	logger.Debug("Request URL: " + scheme + "://" + r.Host + r.RequestURI)
	for key, values := range r.Header {
		logger.Debug(key + ": " + strings.Join(values, ", "))
	}
	logger.Debug("Request body: " + string(reqBody))
	if err != nil {
		logger.Debug("server: could not read request body:" + err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var imageReview v1alpha1.ImageReview

	err = json.Unmarshal(reqBody, &imageReview)

	if err != nil {
		logger.Debug("Error in JSON data:" + err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// INIT THE CHAIN
	validator := &Validate{}
	replicator := &Replicate{}
	scanner := &Scan{}

	if (REPLICATOR_ON) {
			validator.setNext(replicator)
	}
	if (SCANNER_ON) {
		if (REPLICATOR_ON) {
			replicator.setNext(scanner)
		} else {
			validator.setNext(scanner)
		}
	}

	// BEGIN THE CHAIN
	validator.execute(&imageReview)

	// Return Response
	jsonData, _ := json.Marshal(imageReview)
	if (imageReview.Status.Allowed == true) {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
		return
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(jsonData)
		return	
	}
}

func main() {
	http.HandleFunc("/ipw", imagePolicyWebhookHandler)
	// Required env vars
	envVarsExists := []string{"PORT", "WHITE_LIST"}
	envVars := make(map[string]string)
	notFoundVars := []string{}
	for _, env := range envVarsExists {
		el, exists := os.LookupEnv(env)
		if !exists {
			notFoundVars = append(notFoundVars, env)
		} else {
			envVars[env] = el
		}
	}
	if len(notFoundVars) != 0 {
		logger.Info( strings.Join(notFoundVars, ", ") + " variables are not found! MUST fill these enviroment variables!")
		return
	}

	// GET WHITELIST
	WHITELIST = strings.Split(envVars["WHITE_LIST"], ",")
	for i, el := range WHITELIST {
		el = strings.Replace(el, " ", "", -1)
		WHITELIST[i] = el
	}

	// GET EXCLUDED NAMESPACES
	exclude_list := os.Getenv("EXCLUDE_NAMESPACES")
	EXCLUDE_NAMESPACES = strings.Split(exclude_list, ",")
	for i, el := range EXCLUDE_NAMESPACES {
		el = strings.Replace(el, " ", "", -1)
		EXCLUDE_NAMESPACES[i] = el
	}
	// Check replicator env vars
	replicator_string, replicator_exists := os.LookupEnv("REPLICATOR_ON")
	if (replicator_exists) {
		on, err := strconv.ParseBool(replicator_string)
		if err != nil {
			panic(err)
		}
		REPLICATOR_ON = on
		getReplicatorEnv()
	}
	// Check scanner env vars
	scanner_string, scanner_exists := os.LookupEnv("SCANNER_ON")
	if (scanner_exists) {
		on, err := strconv.ParseBool(scanner_string)
		if err != nil {
			panic(err)
		}
		SCANNER_ON = on
	}

	fmt.Printf("Starting server at port %s\n", envVars["PORT"])

	if os.Getenv("DEBUG") == "false" {
		logger = zap.Must(zap.NewProduction())
	}
	defer logger.Sync()
	logger.Info("Logger initialized")
	logger.Debug("WHITELIST: " + strings.Join(WHITELIST, " | "))
	logger.Debug("EXCLUDED NAMESPACES: " + strings.Join(EXCLUDE_NAMESPACES, " | "))

	if err := http.ListenAndServe(":" + string(envVars["PORT"]), nil); err != nil {
		logger.Fatal(err.Error())
	}
}