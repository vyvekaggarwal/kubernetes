/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package windows

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	imageutils "k8s.io/kubernetes/test/utils/image"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	emptyDirVolumePath = "C:\\test-volume"
	hostMapPath        = "C:\\tmp"
	containerName      = "test-container"
	volumeName         = "test-volume"
)

var (
	image = imageutils.GetE2EImage(imageutils.Pause)
)

var _ = SIGDescribe("Windows volume mounts ", func() {
	f := framework.NewDefaultFramework("windows-volumes")
	var (
		emptyDirSource = v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{
				Medium: v1.StorageMediumDefault,
			},
		}

		hostMapSource = v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: hostMapPath,
			},
		}
	)
	BeforeEach(func() {
		framework.SkipUnlessNodeOSDistroIs("windows")
	})

	Context("check volume mount permissions", func() {

		It("container should have readOnly permissions on emptyDir", func() {

			By("creating a container with readOnly permissions on emptyDir volume")
			doReadOnlyTest(f, emptyDirSource, emptyDirVolumePath)

			By("creating two containers, one with readOnly permissions the other with read-write permissions on emptyDir volume")
			doReadWriteReadOnlyTest(f, emptyDirSource, emptyDirVolumePath)
		})

		It("container should have readOnly permissions on hostMapPath", func() {

			By("creating a container with readOnly permissions on hostMap volume")
			doReadOnlyTest(f, hostMapSource, hostMapPath)

			By("creating two containers, one with readOnly permissions the other with read-write permissions on hostMap volume")
			doReadWriteReadOnlyTest(f, hostMapSource, hostMapPath)
		})

	})

})

func doReadOnlyTest(f *framework.Framework, source v1.VolumeSource, volumePath string) {
	var (
		filePath = volumePath + "\\test-file.txt"
		podName  = "pod-" + string(uuid.NewUUID())
		pod      = testPodWithROVolume(podName, source, volumePath)
	)

	f.PodClient().CreateSync(pod)
	cmd := []string{"cmd", "/c", "echo windows-volume-test", ">", filePath}

	_, stderr, _ := f.ExecCommandInContainerWithFullOutput(podName, containerName, cmd...)

	Expect(stderr).To(Equal("Access is denied."))

}

func doReadWriteReadOnlyTest(f *framework.Framework, source v1.VolumeSource, volumePath string) {
	var (
		filePath        = volumePath + "\\test-file"
		podName         = "pod-" + string(uuid.NewUUID())
		pod             = testPodWithROVolume(podName, source, volumePath)
		rwcontainerName = containerName + "-rw"
	)

	rwcontainer := v1.Container{
		Name:  containerName + "-rw",
		Image: image,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: volumePath,
			},
		},
	}

	pod.Spec.Containers = append(pod.Spec.Containers, rwcontainer)
	f.PodClient().CreateSync(pod)

	cmd := []string{"cmd", "/c", "echo windows-volume-test", ">", filePath}

	stdout_rw, stderr_rw, err_rw := f.ExecCommandInContainerWithFullOutput(podName, rwcontainerName, cmd...)
	msg := fmt.Sprintf("cmd: %v, stdout: %q, stderr: %q", cmd, stdout_rw, stderr_rw)
	Expect(err_rw).NotTo(HaveOccurred(), msg)

	_, stderr, _ := f.ExecCommandInContainerWithFullOutput(podName, containerName, cmd...)
	Expect(stderr).To(Equal("Access is denied."))

	readcmd := []string{"cmd", "/c", "type", filePath}
	readout, readerr, err := f.ExecCommandInContainerWithFullOutput(podName, containerName, readcmd...)
	readmsg := fmt.Sprintf("cmd: %v, stdout: %q, stderr: %q", readcmd, readout, readerr)
	Expect(readout).To(Equal("windows-volume-test"))
	Expect(err).NotTo(HaveOccurred(), readmsg)
}

func testPodWithROVolume(podName string, source v1.VolumeSource, path string) *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  containerName,
					Image: image,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: path,
							ReadOnly:  true,
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{
				{
					Name:         volumeName,
					VolumeSource: source,
				},
			},
		},
	}
}
