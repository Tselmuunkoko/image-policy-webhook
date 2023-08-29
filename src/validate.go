package main

import (
	"k8s.io/api/imagepolicy/v1alpha1"
	"regexp"
	"fmt"
	"strings"
)

type Validate struct {
    next ImagePolicyWebhook
}

func (v *Validate) execute(i *v1alpha1.ImageReview) {
	validate(i)
	v.next.execute(i)
}

func (v *Validate) setNext(next ImagePolicyWebhook) {
    v.next = next
}

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

func validate(imageReview *v1alpha1.ImageReview) (resultReview *v1alpha1.ImageReview) {
	resultReview = imageReview
	if isExcluded(imageReview.Spec.Namespace) {
		resultReview.Status.Allowed = true
		logger.Info("Passed the requirement")
		return
	}
	var images = false
	for _, container := range imageReview.Spec.Containers {
		registry, project, image, tag, hash, err := splitDockerImage(container.Image)
		if err != nil {
			logger.Info("Couldn't pass the requirement")
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
			return
		}
	}
	resultReview.Status.Allowed = images
	logger.Info("Passed the requirement")
	return
}