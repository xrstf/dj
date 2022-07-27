package prow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodIdentifier struct {
	BuildID string
	JobID   string
}

func ParsePodIdentifier(arg string) (*PodIdentifier, error) {
	// is it a UUID?
	parsed, err := uuid.Parse(arg)
	if err == nil {
		return &PodIdentifier{JobID: parsed.String()}, nil
	}

	// arg doesn't look like a UUID, maybe it's a valid build ID instead?
	if _, err := strconv.ParseInt(arg, 10, 64); err == nil {
		return &PodIdentifier{BuildID: arg}, nil
	}

	return nil, fmt.Errorf("%q is neither a UUID nor a valid 64-bit build ID", arg)
}

func (i *PodIdentifier) LabelSelector() string {
	if i.BuildID != "" {
		return fmt.Sprintf("prow.k8s.io/build-id=%s", i.BuildID)
	}

	if i.JobID != "" {
		return fmt.Sprintf("prow.k8s.io/id=%s", i.JobID)
	}

	return ""
}

func (i *PodIdentifier) Fields() logrus.Fields {
	fields := logrus.Fields{}
	if i.BuildID != "" {
		fields["build"] = i.BuildID
	}

	if i.JobID != "" {
		fields["job"] = i.JobID
	}

	return fields
}

type PodCheckerFunc func(pod *corev1.Pod) bool

func (i *PodIdentifier) WaitForPod(ctx context.Context, clientset *kubernetes.Clientset, namespace string, validPod PodCheckerFunc, giveUp PodCheckerFunc) (*corev1.Pod, error) {
	wi, err := clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: i.LabelSelector(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to watch Pods: %w", err)
	}

	if giveUp == nil {
		giveUp = func(_ *corev1.Pod) bool {
			return false
		}
	}

	var pod *corev1.Pod

	for event := range wi.ResultChan() {
		if pod != nil {
			continue
		}

		var ok bool
		pod, ok = event.Object.(*corev1.Pod)
		if !ok {
			pod = nil
			continue
		}

		if !validPod(pod) && !giveUp(pod) {
			pod = nil
			continue
		}

		// we found what we were looking for, stop generating new events
		wi.Stop()

		// let the loop finish, in case there are more events (that we will ignore)
	}

	if pod == nil || !validPod(pod) {
		return nil, nil
	}

	return pod, nil
}
