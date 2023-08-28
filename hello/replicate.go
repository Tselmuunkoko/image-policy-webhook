package main

import (
	"k8s.io/api/imagepolicy/v1alpha1"
)

func replicate(imageReview *v1alpha1.ImageReview) (resultReview *v1alpha1.ImageReview)  {
	// pull docker
	logger.Info("Replication")
	resultReview = imageReview
	return
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
