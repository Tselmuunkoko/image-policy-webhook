package main

import (
	"k8s.io/api/imagepolicy/v1alpha1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"context"
	"io"
	"os"
	"encoding/json"
	"encoding/base64"
	"strings"
	"bytes"
)

func replicate(imageReview *v1alpha1.ImageReview) (resultReview *v1alpha1.ImageReview)  {
	// pull docker
	logger.Info("Replication started!")
	resultReview = imageReview
	reason := make(map[string]string)
	cli, err := client.NewClientWithOpts(client.WithHost(REPLICATOR_ENV_VARS["APP_DOCKER_HOST"]))
	if err != nil {
		panic(err)
	}
	
	pulled_images := []string{}

	// AUTHENTICATE
	authConfig := types.AuthConfig{
		Username: REPLICATOR_ENV_VARS["PRIVATE_REGISTRY_USERNAME"],
		Password: REPLICATOR_ENV_VARS["PRIVATE_REGISTRY_PASSWORD"],
	}
	
	authStr, err := encodeAuth(authConfig)
	if err != nil {
		logger.Error("Error encoding auth config:" + err.Error())
		return
	}
	// PULL
	prefix := REPLICATOR_ENV_VARS["PRIVATE_REGISTRY_HOST"]
	namespace := REPLICATOR_ENV_VARS["PRIVATE_REGISTRY_NAMESPACE"]

	for _, container := range imageReview.Spec.Containers {
		if !strings.HasPrefix(container.Image, prefix) {
			logger.Info("Trying pull " + container.Image + " image.")
			out, err := cli.ImagePull(context.Background(), container.Image, types.ImagePullOptions{})
			if err != nil {
				panic(err)
			}
			defer out.Close()

			// Print the pull progress
			_, err = io.Copy(os.Stdout, out)
			if err != nil {
				logger.Error("Error copying image pull output:" + err.Error())
				reason[container.Image] = "Couldn't pull this image"
			} else {
				pulled_images = append(pulled_images, container.Image)
				logger.Info(container.Image + " pulled successfully!")
			}
		}
	}
	// TAG
	tagged_images := []string{}
	for _, p_image := range pulled_images {
		logger.Info("Tagging pulled image "+ p_image)
		_, _, image, tag, _, err := splitDockerImage(p_image)
		logger.Info("image: " + image)
		logger.Info("tag: " + tag)
		newImageTag := prefix + "/" + namespace + "/" + image + ":" + tag
		err = cli.ImageTag(context.Background(), p_image, newImageTag)
		if err != nil {
			logger.Error("Error tagging image:" + err.Error())
		} else {
			tagged_images = append(tagged_images, newImageTag)
			logger.Info(p_image +" tagged into "+ newImageTag +" successfully!")
		}
	}

	// PUSH
	for _, t_image := range tagged_images {
		pushOpts := types.ImagePushOptions{
			RegistryAuth: authStr,
		}
		pushOut, err := cli.ImagePush(context.Background(), t_image, pushOpts)
		defer pushOut.Close()
		if err != nil {
			panic(err)
		}
		var outputBuffer bytes.Buffer
		_, err = io.Copy(io.MultiWriter(&outputBuffer, os.Stdout), pushOut)
		if err != nil {
			logger.Error("Error pushing image:" + err.Error())
		}

		capturedOutput := outputBuffer.String()

		if strings.Contains(capturedOutput, "error") {
			reason[t_image] = "replication failed while pushing this image!"
		} else {
			reason[t_image] = "pushed this tagged image into private registry."
		}
	}

	// REMOVE
	for _, p_image := range pulled_images {
		_, err := cli.ImageRemove(context.Background(), p_image, types.ImageRemoveOptions{})
		if err != nil {
			logger.Error("Error removing image:" + err.Error())
		} else {
			logger.Info(p_image + " removed successfully!")
		}
	}
	jsonReason, _ := json.Marshal(reason)
	resultReview.Status.Reason = string(jsonReason)
	return
}

func encodeAuth(authConfig types.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

type Replicate struct {
    next ImagePolicyWebhook
}

func (r *Replicate) execute(i *v1alpha1.ImageReview) {
	replicate(i)
	r.next.execute(i)
}

func (r *Replicate) setNext(next ImagePolicyWebhook) {
    r.next = next
}

func getReplicatorEnv() {
	replicatorEnvVarsExists := []string{
		"PRIVATE_REGISTRY_USERNAME",
		"PRIVATE_REGISTRY_PASSWORD",
		"PRIVATE_REGISTRY_NAMESPACE",
		"PRIVATE_REGISTRY_HOST",
	}
	replicatorNotFoundVars := []string{}
	for _, env := range replicatorEnvVarsExists {
		el, exists := os.LookupEnv(env)
		if !exists {
			replicatorNotFoundVars = append(replicatorNotFoundVars, env)
		} else {
			REPLICATOR_ENV_VARS[env] = envClean(el)
		}
	}
	if len(replicatorNotFoundVars) != 0 {
		logger.Info( strings.Join(replicatorNotFoundVars, ", ") + " variables are not found! MUST fill these enviroment variables for REPLICATOR!")
		return
	}
	REPLICATOR_ENV_VARS["APP_DOCKER_HOST"] = os.Getenv("APP_DOCKER_HOST"); if REPLICATOR_ENV_VARS["APP_DOCKER_HOST"] == "" {REPLICATOR_ENV_VARS["APP_DOCKER_HOST"] = "unix:///var/run/docker.sock"}
}

func envClean(input string) (cleaned string){
	cleaned = strings.ReplaceAll(input, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, "\"", "")
	return cleaned
}