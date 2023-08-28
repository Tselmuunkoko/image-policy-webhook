package main

import (
	"k8s.io/api/imagepolicy/v1alpha1"
)

func scan(imageReview *v1alpha1.ImageReview) (resultReview *v1alpha1.ImageReview) {
	// scan
	logger.Info("Scan")
	resultReview = imageReview
	return
}


type Scan struct {
    next ImagePolicyWebhook
}

func (s *Scan) execute(i *v1alpha1.ImageReview) {
	scan(i)
}

func (s *Scan) setNext(next ImagePolicyWebhook) {
    s.next = next
}