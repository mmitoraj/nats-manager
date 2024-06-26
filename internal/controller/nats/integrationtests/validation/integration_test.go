package validation_test

import (
	"log"
	"os"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/kyma-project/nats-manager/testutils/integration"
	nmtsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
	"github.com/onsi/gomega"
	onsigomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const projectRootDir = "../../../../../"

const noError = ""

const (
	spec           = "spec"
	kind           = "kind"
	cluster        = "cluster"
	jetStream      = "jetStream"
	memStorage     = "memStorage"
	fileStorage    = "fileStorage"
	apiVersion     = "apiVersion"
	logging        = "logging"
	metadata       = "metadata"
	name           = "name"
	namespace      = "namespace"
	kindNATS       = "NATS"
	size           = "size"
	enabled        = "enabled"
	apiVersionNATS = "operator.kyma-project.io/v1alpha1"
)

var testEnvironment *integration.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(projectRootDir, true, nil)
	if err != nil {
		log.Fatal(err)
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err = testEnvironment.TearDown(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func Test_Validate_CreateNATS(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		// We use Unstructured instead of NATS to ensure that all undefined properties are nil and not Go defaults.
		givenUnstructuredNATS unstructured.Unstructured
		wantErrMsg            string
	}{
		{
			name: `validation of spec.cluster.size passes for odd numbers`,
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						cluster: map[string]any{
							size: 3,
						},
					},
				},
			},
			wantErrMsg: noError,
		},
		{
			name: `validation of spec.cluster.size fails for even numbers`,
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						cluster: map[string]any{
							size: 4,
						},
					},
				},
			},
			wantErrMsg: "size only accepts odd numbers",
		},
		{
			name: `validation of spec.cluster.size fails for numbers < 1`,
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						cluster: map[string]any{
							size: -1,
						},
					},
				},
			},
			wantErrMsg: "should be greater than or equal to 1",
		},
		{
			name: `validation of spec.jetStream.memStorage passes if enabled is true and size is not 0`,
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{
							memStorage: map[string]any{
								enabled: true,
								size:    "1Gi",
							},
						},
					},
				},
			},
			wantErrMsg: noError,
		},
		{
			name: `validation of spec.jetStream.memStorage passes if size is 0 but enabled is false`,
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{
							memStorage: map[string]any{
								enabled: false,
								size:    0,
							},
						},
					},
				},
			},
			wantErrMsg: noError,
		},
		{
			name: `validation of spec.jetStream.memStorage fails if enabled is true but size is 0`,
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{
							memStorage: map[string]any{
								enabled: true,
								size:    0,
							},
						},
					},
				},
			},
			wantErrMsg: "can only be enabled if size is not 0",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenUnstructuredNATS.GetNamespace())

			// when
			err := testEnvironment.CreateUnstructuredK8sResource(&tc.givenUnstructuredNATS)

			// then
			if tc.wantErrMsg == noError {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.wantErrMsg)
			}
		})
	}
}

// Test_Validate_UpdateNATS creates a givenNATS on a K8s cluster, runs wantMatches against the corresponding NATS
// object in the K8s cluster, then tries to modify it with givenUpdates, and test the error that was caused by this
// update, against a wantErrMsg.
func Test_Validate_UpdateNATS(t *testing.T) {
	testCases := []struct {
		name         string
		givenNATS    *nmapiv1alpha1.NATS
		wantMatches  onsigomegatypes.GomegaMatcher
		givenUpdates []testutils.NATSOption
		wantErrMsg   string
	}{
		{
			name: `validation of fileStorage fails, if fileStorage.size gets changed`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSFileStorage(defaultFileStorage()),
			),
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecJetStreamFileStorage(defaultFileStorage()),
			),
			givenUpdates: []testutils.NATSOption{
				testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
					StorageClassName: defaultFileStorage().StorageClassName,
					Size:             resource.MustParse("2Gi"),
				}),
			},
			wantErrMsg: "fileStorage is immutable once it was set",
		},
		{
			name: `validation of fileStorage fails, if fileStorage.storageClassName gets changed`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSFileStorage(defaultFileStorage()),
			),
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecJetStreamFileStorage(defaultFileStorage()),
			),
			givenUpdates: []testutils.NATSOption{
				testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
					StorageClassName: "not-standard",
					Size:             defaultFileStorage().Size,
				}),
			},
			wantErrMsg: "fileStorage is immutable once it was set",
		},
		{
			name: `validation of cluster fails, if cluster.size was set to a value >1 and now gets reduced to 1'`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCluster(defaultCluster()),
			),
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecCluster(defaultCluster()),
			),
			givenUpdates: []testutils.NATSOption{
				testutils.WithNATSCluster(nmapiv1alpha1.Cluster{
					Size: 1,
				}),
			},
			wantErrMsg: "cannot be set to 1 if size was greater than 1",
		},
		{
			name: `validation of cluster passes, if cluster.size was set to a value >1
			and now gets set to another value >1'`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCluster(defaultCluster()),
			),
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecCluster(defaultCluster()),
			),
			givenUpdates: []testutils.NATSOption{
				testutils.WithNATSCluster(nmapiv1alpha1.Cluster{
					Size: 5,
				}),
			},
			wantErrMsg: noError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenNATS.GetNamespace())

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// then
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)

			// when
			var err error
			require.Eventually(t, func() bool {
				err = testEnvironment.UpdatedNATSInK8s(tc.givenNATS, tc.givenUpdates...)
				return !kapierrors.IsConflict(err)
			}, integration.SmallTimeOut, integration.SmallPollingInterval, "failed to update NATS CR")

			// then
			if tc.wantErrMsg == noError {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.wantErrMsg)
			}
		})
	}
}

func Test_NATS_Defaulting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		// We use Unstructured instead of NATS to ensure that all undefined properties are nil and not Go defaults.
		givenUnstructuredNATS unstructured.Unstructured
		wantMatches           onsigomegatypes.GomegaMatcher
	}{
		{
			name: "defaulting with bare minimum NATS",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
				},
			},
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecCluster(defaultCluster()),
				nmtsmatchers.HaveSpecResources(defaultResources()),
				nmtsmatchers.HaveSpecLogging(defaultLogging()),
				nmtsmatchers.HaveSpecJetsStreamMemStorage(defaultMemStorage()),
				nmtsmatchers.HaveSpecJetStreamFileStorage(defaultFileStorage()),
			),
		},
		{
			name: "defaulting with an empty spec",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{},
				},
			},
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecCluster(defaultCluster()),
				nmtsmatchers.HaveSpecResources(defaultResources()),
				nmtsmatchers.HaveSpecLogging(defaultLogging()),
				nmtsmatchers.HaveSpecJetsStreamMemStorage(defaultMemStorage()),
				nmtsmatchers.HaveSpecJetStreamFileStorage(defaultFileStorage()),
			),
		},
		{
			name: "defaulting with an empty spec.cluster",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						cluster: map[string]any{},
					},
				},
			},
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecCluster(defaultCluster()),
			),
		},
		{
			name: "defaulting with an empty spec.jetStream",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{},
					},
				},
			},
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecJetsStreamMemStorage(defaultMemStorage()),
				nmtsmatchers.HaveSpecJetStreamFileStorage(defaultFileStorage()),
			),
		},
		{
			name: "defaulting with an empty spec.jetStream.memStorage and spec.jetStream.fileStorage",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{
							memStorage:  map[string]any{},
							fileStorage: map[string]any{},
						},
					},
				},
			},
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecJetsStreamMemStorage(defaultMemStorage()),
				nmtsmatchers.HaveSpecJetStreamFileStorage(defaultFileStorage()),
			),
		},
		{
			name: "defaulting with an empty spec.logging",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{},
					},
				},
			},
			wantMatches: gomega.And(
				nmtsmatchers.HaveSpecLogging(defaultLogging()),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenUnstructuredNATS.GetNamespace())

			// when
			testEnvironment.EnsureK8sUnStructResourceCreated(t, &tc.givenUnstructuredNATS)

			// then
			testEnvironment.GetNATSAssert(g, &nmapiv1alpha1.NATS{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      tc.givenUnstructuredNATS.GetName(),
					Namespace: tc.givenUnstructuredNATS.GetNamespace(),
				},
			}).Should(tc.wantMatches)
		})
	}
}

func defaultResources() kcorev1.ResourceRequirements {
	return kcorev1.ResourceRequirements{
		Limits: kcorev1.ResourceList{
			"cpu":    resource.MustParse("500m"),
			"memory": resource.MustParse("1Gi"),
		},
		Requests: kcorev1.ResourceList{
			"cpu":    resource.MustParse("40m"),
			"memory": resource.MustParse("64Mi"),
		},
	}
}

func defaultMemStorage() nmapiv1alpha1.MemStorage {
	return nmapiv1alpha1.MemStorage{
		Enabled: true,
		Size:    resource.MustParse("1Gi"),
	}
}

func defaultFileStorage() nmapiv1alpha1.FileStorage {
	return nmapiv1alpha1.FileStorage{
		StorageClassName: "default",
		Size:             resource.MustParse("1Gi"),
	}
}

func defaultLogging() nmapiv1alpha1.Logging {
	return nmapiv1alpha1.Logging{
		Debug: false,
		Trace: false,
	}
}

func defaultCluster() nmapiv1alpha1.Cluster {
	return nmapiv1alpha1.Cluster{Size: 3}
}
