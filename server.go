package main

import (
    "fmt"
    "log"
    "net/http"
	"io/ioutil"
	"k8s.io/api/imagepolicy/v1alpha1"
	"encoding/json"
	"regexp"
	// "sort"
)

const dockerImagePattern = `^(?P<registry>.+?\/)?(?P<project>[\w\-]+)\/(?P<image>[\w\-]+):(?P<tag>[\w\-.]+)$`
const dockerImageHashPattern =`^(?P<registry>.+?\/)?(?P<project>[\w\-]+)\/(?P<image>[\w\-]+)@(?P<hash>sha256:[\w]+)?$`

// func checkWhiteList(registry string) (result bool) {
// 	whitelist := []string{'harbor.it.org'}
// 	index := sort.SearchStrings(whitelist, registry)
// 	if index < len(whitelist) && whitelist[index] == registry {
// 		result = true
// 		return 
// 	} else {
// 		result = false
// 		return
// 	}
// }
func splitDockerImage(imageStr string) (registry, project, image, tag string, hash string, err error) {
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
    if r.URL.Path != "/hello" {
        http.Error(w, "404 not found.", http.StatusNotFound)
        return
    }

    if (r.Method == "GET" || r.Method == "POST") {
        reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("server: could not read request body: %s\n", err)
		}
		var imageReview v1alpha1.ImageReview
		var resultReview v1alpha1.ImageReview
		err = json.Unmarshal(reqBody, &imageReview)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		for _, container := range imageReview.Spec.Containers {
			registry, project, image, tag, hash, err := splitDockerImage(container.Image)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			fmt.Println("Registry:", registry)
			fmt.Println("Project:", project)
			fmt.Println("Image:", image)
			fmt.Println("Tag:", tag)
			fmt.Println("Hash:", hash)
			// if tag == nil || tag == 'latest' || checkWhiteList(registry) {
			// 	resultReview.Status.Allowed = false
			// 	fmt.Fprintf(w, json.marshal(resultReview))
			// }
		}
		// fmt.Fprintf(w, "h")
		resultReview.Status.Allowed = true
		jsonData, err := json.Marshal(resultReview)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Fprint(w, string(jsonData), http.StatusOK)
    } else {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
	}
}


func main() {
    http.HandleFunc("/hello", helloHandler) // Update this line of code


    fmt.Printf("Starting server at port 8080\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
}