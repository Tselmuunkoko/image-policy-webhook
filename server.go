package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"regexp"
	"io/ioutil"
	"k8s.io/api/imagepolicy/v1alpha1"
	"os"
	"go.uber.org/zap"
	"strings"
)

var logger = zap.Must(zap.NewDevelopment())
var WHITELIST = []string{}
var EXCLUDE_NAMESPACES = []string{}

const (
	dockerImagePattern     = `^(?P<registry>.+?\/)?(?P<project>[\w\-]+)\/(?P<image>[\w\-]+):(?P<tag>[\w\-.]+)$`
	dockerImageHashPattern = `^(?P<registry>.+?\/)?(?P<project>[\w\-]+)\/(?P<image>[\w\-]+)@(?P<hash>sha256:[\w]+)?$`
)

func checkWhiteList(registry string) bool {
	registry = strings.Replace(registry, "/", "", -1)
	for _, el := range WHITELIST {
		if el == registry {
			return false
		}
	}
	return true
}

func isExcluded(namespace string) bool {
	for _, el := range EXCLUDE_NAMESPACES {
		if el == namespace {
			return true
		}
	}
	return false
}

func splitDockerImage(imageStr string) (registry, project, image, tag, hash string, err error) {
	re := regexp.MustCompile(dockerImagePattern)
	re1 := regexp.MustCompile(dockerImageHashPattern)
	matches := re.FindStringSubmatch(imageStr)
	matches1 := re1.FindStringSubmatch(imageStr)

	if len(matches) != 0 {
		matchMap := make(map[string]string)
		for i, name := range re.SubexpNames() {
			if i != 0 && name != "" {
				matchMap[name] = matches[i]
			}
		}
		registry = matchMap["registry"]
		project = matchMap["project"]
		image = matchMap["image"]
		tag = matchMap["tag"]
	} else if len(matches1) != 0 {
		matchMap := make(map[string]string)
		for i, name := range re1.SubexpNames() {
			if i != 0 && name != "" {
				matchMap[name] = matches1[i]
			}
		}
		registry = matchMap["registry"]
		project = matchMap["project"]
		image = matchMap["image"]
		hash = matchMap["hash"]
	} else {
		err = fmt.Errorf("invalid image format")
	}
	return
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path != "/hello" {
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

	var resultReview v1alpha1.ImageReview
	if isExcluded(imageReview.Spec.Namespace) {
		resultReview.Status.Allowed = true
		jsonData, _ := json.Marshal(resultReview)
		logger.Info("Passed the requirement")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
		return
	}
	var images = false
	for _, container := range imageReview.Spec.Containers {
		registry, project, image, tag, hash, err := splitDockerImage(container.Image)
		if err != nil {
			logger.Info("Couldn't pass the requirement")
			w.WriteHeader(http.StatusOK)
			jsonData, _ := json.Marshal(resultReview)
			w.Write(jsonData)
			return
		}
		images = true
		logger.Debug("Registry:" + registry)
		logger.Debug("Project:" + project)
		logger.Debug("Image:" + image)
		logger.Debug("Tag:" + tag)
		logger.Debug("Hash:" + hash)
		if tag == "" || tag == "latest" || checkWhiteList(registry) {
			logger.Info("Couldn't pass the requirement")
			w.WriteHeader(http.StatusOK)
			jsonData, _ := json.Marshal(resultReview)
			w.Write(jsonData)
			return
		}
	}

	resultReview.Status.Allowed = images
	jsonData, err := json.Marshal(resultReview)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	logger.Info("Passed the requirement")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	// get port
	port, port_exists := os.LookupEnv("PORT")
	if !port_exists {
		logger.Error("Port variable not found")
		return
	}
	// GET WHITELIST
	list, list_exists := os.LookupEnv("WHITE_LIST")
	if !list_exists {
		logger.Error("White list variable not found")
		return
	}
	WHITELIST = strings.Split(list, ",")
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
	fmt.Printf("Starting server at port %s\n", port)

	if os.Getenv("DEBUG") == "false" {
		logger = zap.Must(zap.NewProduction())
	}
	defer logger.Sync()
	logger.Info("Logger initialized")
	logger.Debug("WHITELIST: " + strings.Join(WHITELIST, " | "))
	logger.Debug("EXCLUDED NAMESPACES: " + strings.Join(EXCLUDE_NAMESPACES, " | "))

	if err := http.ListenAndServe(":" + string(port), nil); err != nil {
		logger.Fatal(err.Error())
	}
}