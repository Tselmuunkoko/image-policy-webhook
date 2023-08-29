package main

import (
	"k8s.io/api/imagepolicy/v1alpha1"
)

type ImagePolicyWebhook interface {
    execute(*v1alpha1.ImageReview)
    setNext(ImagePolicyWebhook)
}
