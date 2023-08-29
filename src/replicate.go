package main

import (
	"k8s.io/api/imagepolicy/v1alpha1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"context"
	"io"
	"os"
	"fmt"
	"encoding/json"
	"encoding/base64"
)

func replicate(imageReview *v1alpha1.ImageReview) (resultReview *v1alpha1.ImageReview)  {
	// pull docker
	logger.Info("Replication")
	resultReview = imageReview

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	out, err := cli.ImagePull(context.Background(), "alpine", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// Print the pull progress
	_, err = io.Copy(os.Stdout, out)
	if err != nil {
		fmt.Println("Error copying image pull output:", err)
		return
	}

	fmt.Println("Image pulled successfully!")
	// AUTHENTICATE
	authConfig := types.AuthConfig{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD"),
	}
	
	authStr, err := encodeAuth(authConfig)
	if err != nil {
		fmt.Println("Error encoding auth config:", err)
		return
	}

	// TAG
	newImageTag := os.Getenv("DOCKER_HOST")+"/alpine:3.18"

	// Tag the pulled image with the new tag
	err = cli.ImageTag(context.Background(), "alpine", newImageTag)
	if err != nil {
		fmt.Println("Error tagging image:", err)
		return
	}

	// PUSH
	pushOpts := types.ImagePushOptions{
		RegistryAuth: authStr,
	}
	pushOut, err := cli.ImagePush(context.Background(), newImageTag, pushOpts)
	if err != nil {
		fmt.Println("Error pushing image:", err)
		return
	}
	defer pushOut.Close()

	fmt.Println("Image pushed successfully!")
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
