package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"regexp"
	"sort"
	"io/ioutil"
	"k8s.io/api/imagepolicy/v1alpha1"
)

const (
	dockerImagePattern     = `^(?P<registry>.+?\/)?(?P<project>[\w\-]+)\/(?P<image>[\w\-]+):(?P<tag>[\w\-.]+)$`
	dockerImageHashPattern = `^(?P<registry>.+?\/)?(?P<project>[\w\-]+)\/(?P<image>[\w\-]+)@(?P<hash>sha256:[\w]+)?$`
)

var whitelist = []string{"harbor.it.org/"}

func checkWhiteList(registry string) bool {
	index := sort.SearchStrings(whitelist, registry)
	return !(index < len(whitelist) && whitelist[index] == registry)
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
	w.WriteHeader(http.StatusOK)

	if r.URL.Path != "/hello" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("server: could not read request body: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var imageReview v1alpha1.ImageReview
	err = json.Unmarshal(reqBody, &imageReview)
	if err != nil {
		log.Println("Error in JSON data:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var resultReview v1alpha1.ImageReview
	var images = false
	for _, container := range imageReview.Spec.Containers {
		registry, project, image, tag, hash, err := splitDockerImage(container.Image)
		if err != nil {
			log.Println("Error:", err)
			fmt.Println("Couldn't pass the requirement")
			w.WriteHeader(http.StatusOK)
			jsonData, _ := json.Marshal(resultReview)
			w.Write(jsonData)
			return
		}
		images = true
		fmt.Println("Registry:", registry)
		fmt.Println("Project:", project)
		fmt.Println("Image:", image)
		fmt.Println("Tag:", tag)
		fmt.Println("Hash:", hash)
		if tag == "" || tag == "latest" || checkWhiteList(registry) {
			fmt.Println("Couldn't pass the requirement")
			w.WriteHeader(http.StatusOK)
			jsonData, _ := json.Marshal(resultReview)
			w.Write(jsonData)
			return
		}
	}

	resultReview.Status.Allowed = images
	jsonData, err := json.Marshal(resultReview)
	if err != nil {
		log.Println("Error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
