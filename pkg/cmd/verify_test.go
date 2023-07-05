package cmd_test

import (
	"errors"
	"strings"

	"github.com/cnoe-io/cnoe-cli/pkg/cmd"
	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	"github.com/cnoe-io/cnoe-cli/pkg/lib/libfakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Verify", func() {

	var (
		fakeK8sClient *libfakes.FakeIK8sClient
		cfg           *[]lib.Config
		stdout        *gbytes.Buffer
		stderr        *gbytes.Buffer
	)

	BeforeEach(func() {
		stdout = gbytes.NewBuffer()
		// stderr = gbytes.NewBuffer()
		fakeK8sClient = &libfakes.FakeIK8sClient{}
	})

	Context("when GVK is not matching", func() {
		Context("with missing apiVersion", func() {
			BeforeEach(func() {
				cfg = &[]lib.Config{{}}
			})

			It("indicate that apiVersion is not matching", func() {
				err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apiVersion or kind not matching"))
			})
		})

		Context("with missing Kind", func() {
			BeforeEach(func() {
				cfg = &[]lib.Config{{
					ApiVersion: "cnoe.io/v1alpha1",
				}}
			})

			It("indicate that apiVersion is not matching", func() {
				err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("apiVersion or kind not matching"))
			})
		})
	})

	Context("when name is missing", func() {
		BeforeEach(func() {
			cfg = &[]lib.Config{{
				ApiVersion: "cnoe.io/v1alpha1",
				Kind:       "Prerequisite",
			}}
		})

		It("indicate that name is missing", func() {
			err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing metadata.name"))
		})
	})

	Context("when verifying a CRD", func() {
		BeforeEach(func() {
			cfg = &[]lib.Config{
				{
					ApiVersion: "cnoe.io/v1alpha1",
					Kind:       "Prerequisite",
					Metadata: lib.Metadata{
						Name: "test-prereq",
					},
					Spec: lib.Spec{
						Crds: []lib.CRD{
							{
								Group:   "test-group",
								Kind:    "test-kind",
								Version: "test-version",
							},
						},
					},
				},
			}
		})

		Context("when the CRD exists", func() {
			BeforeEach(func() {
				ul := &unstructured.UnstructuredList{}
				ul.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   "test-group",
					Kind:    "test-kind",
					Version: "test-version",
				})
				fakeK8sClient.CRDsReturns(ul, nil)
			})

			It("successfully verifies that CRD exists", func() {
				err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(stdout.Contents())).To(ContainSubstring("✓"))
				Expect(string(stdout.Contents())).To(ContainSubstring("test-group/test-version, Kind=test-kind"))
			})
		})

		Context("when the CRD does not exist", func() {
			BeforeEach(func() {
				fakeK8sClient.CRDsReturns(nil, errors.New("some-error"))
			})

			It("indicate that the CRD does not exist", func() {
				err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
				Expect(err).To(HaveOccurred())

				Expect(string(stdout.Contents())).To(ContainSubstring("X"))
				Expect(string(stdout.Contents())).To(ContainSubstring("test-group/test-version, Kind=test-kind"))
			})
		})
	})

	Context("when verifying a Pod", func() {
		var podList *corev1.PodList
		BeforeEach(func() {
			podList = &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-pod-1",
							Namespace: "ns1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-pod-2",
							Namespace: "ns2",
						},
					},
				},
			}
			fakeK8sClient.PodsReturns(podList, nil)
		})

		Context("when the Pod exists without a namespace", func() {
			BeforeEach(func() {
				cfg = &[]lib.Config{
					{
						ApiVersion: "cnoe.io/v1alpha1",
						Kind:       "Prerequisite",
						Metadata: lib.Metadata{
							Name: "test-prereq",
						},
						Spec: lib.Spec{
							Pods: []lib.Pod{
								{
									Name: "test-pod",
								},
							},
						},
					},
				}
			})

			Context("when Running", func() {
				BeforeEach(func() {
					podList.Items[0].Status.Phase = corev1.PodRunning
					podList.Items[1].Status.Phase = corev1.PodRunning
				})

				It("successfully verifies matching pods", func() {
					err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(stdout.Contents())).To(ContainSubstring("✓"))

					splitString := strings.Split(strings.Trim(string(stdout.Contents()), "\n"), "\n")
					Expect(splitString).To(HaveLen(2))
					Expect(splitString).Should(ConsistOf(
						ContainSubstring("ns1, Pod=test-pod-1 - Running"),
						ContainSubstring("ns2, Pod=test-pod-2 - Running"),
					))
				})
			})

			Context("when Failed", func() {
				BeforeEach(func() {
					podList.Items[0].Status.Phase = corev1.PodFailed
					podList.Items[1].Status.Phase = corev1.PodFailed
				})

				It("indicates that the pod failed", func() {
					err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
					Expect(err).To(HaveOccurred())

					Expect(string(stdout.Contents())).To(ContainSubstring("X"))
					splitString := strings.Split(strings.Trim(string(stdout.Contents()), "\n"), "\n")
					Expect(splitString).To(HaveLen(2))
					Expect(splitString).Should(ConsistOf(
						ContainSubstring("ns1, Pod=test-pod-1 - Failed"),
						ContainSubstring("ns2, Pod=test-pod-2 - Failed"),
					))
				})
			})
		})

		Context("when the Pod exists with a namespace", func() {
			BeforeEach(func() {
				cfg = &[]lib.Config{
					{
						ApiVersion: "cnoe.io/v1alpha1",
						Kind:       "Prerequisite",
						Metadata: lib.Metadata{
							Name: "test-prereq",
						},
						Spec: lib.Spec{
							Pods: []lib.Pod{
								{
									Name:      "test-pod",
									Namespace: "ns1",
								},
							},
						},
					},
				}
			})

			Context("when Running", func() {
				BeforeEach(func() {
					podList.Items[0].Status.Phase = corev1.PodRunning
				})
				It("successfully verifies matching pods", func() {
					err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(stdout.Contents())).To(ContainSubstring("✓"))

					splitString := strings.Split(strings.Trim(string(stdout.Contents()), "\n"), "\n")
					Expect(splitString).To(HaveLen(1))
					Expect(splitString).Should(ConsistOf(ContainSubstring("ns1, Pod=test-pod-1 - Running")))
					Expect(splitString).ShouldNot(ConsistOf(ContainSubstring("ns2, Pod=test-pod-2")))
				})
			})

			Context("when Failed", func() {
				BeforeEach(func() {
					podList.Items[0].Status.Phase = corev1.PodFailed
				})

				It("fails verify", func() {
					err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
					Expect(err).To(HaveOccurred())

					Expect(string(stdout.Contents())).To(ContainSubstring("X"))

					splitString := strings.Split(strings.Trim(string(stdout.Contents()), "\n"), "\n")
					Expect(splitString).To(HaveLen(1))
					Expect(splitString).Should(ConsistOf(ContainSubstring("ns1, Pod=test-pod-1 - Failed")))
					Expect(splitString).ShouldNot(ConsistOf(ContainSubstring("ns2, Pod=test-pod-2")))
				})
			})
		})

		Context("when the Pod does not exist", func() {
			BeforeEach(func() {
				cfg = &[]lib.Config{
					{
						ApiVersion: "cnoe.io/v1alpha1",
						Kind:       "Prerequisite",
						Metadata: lib.Metadata{
							Name: "test-prereq",
						},
						Spec: lib.Spec{
							Pods: []lib.Pod{
								{
									Name: "non-existing-pod",
								},
							},
						},
					},
				}
			})

			It("indicate that the Pod does not exist", func() {
				err := cmd.Verify(stdout, stderr, fakeK8sClient, *cfg)
				Expect(err).To(HaveOccurred())

				Expect(string(stdout.Contents())).To(ContainSubstring("X"))
				Expect(string(stdout.Contents())).To(ContainSubstring("Pod=non-existing-pod"))
			})
		})
	})
})
