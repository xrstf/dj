// SPDX-FileCopyrightText: 2024 Christoph Mewes
// SPDX-License-Identifier: MIT

package util

import (
	"bytes"
	"context"
	"errors"
	"io"

	"go.xrstf.de/dj/pkg/prow"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/term"
)

func RunCommand(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, pod *corev1.Pod, container string, command []string, stdin io.Reader) (string, error) {
	request := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	option := &corev1.PodExecOptions{
		Container: container,
		Command:   command,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
	}

	request.VersionedParams(option, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", request.URL())
	if err != nil {
		return "", err
	}

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return stdout.String(), errors.New(stderr.String())
	}

	return stdout.String(), nil
}

func RunCommandWithTTY(ctx context.Context, clientset *kubernetes.Clientset, restConfig *rest.Config, pod *corev1.Pod, container string, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	terminal := term.TTY{
		In:  stdin,
		Out: stdout,
		Raw: true, // we want stdin attached and a TTY
	}

	var sizeQueue remotecommand.TerminalSizeQueue
	if terminal.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = terminal.MonitorSize(terminal.GetSize())
	}

	return terminal.Safe(func() error {
		request := clientset.CoreV1().RESTClient().
			Post().
			Resource("pods").
			Name(pod.Name).
			Namespace(pod.Namespace).
			SubResource("exec")

		option := &corev1.PodExecOptions{
			Container: prow.TestContainerName,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       true,
		}

		request.VersionedParams(option, scheme.ParameterCodec)
		exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", request.URL())
		if err != nil {
			return err
		}

		return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdin,
			Stdout:            stdout,
			Stderr:            stderr,
			Tty:               true,
			TerminalSizeQueue: sizeQueue,
		})
	})
}
