package framework

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/aws/eks-anywhere/internal/pkg/api"
)

type MulticlusterE2ETest struct {
	T                 *testing.T
	ManagementCluster *ClusterE2ETest
	WorkloadClusters  WorkloadClusters
	// MaxConcurrentWorkers defines the max number of workers for concurrent operations.
	// If it's -1, it will use one worker per job.
	MaxConcurrentWorkers int
	// PersistentManagementCluster avoids creating a management cluster if it finds a kubeconfig
	// in the cluster folder. Useful for local development of tests.
	PersistentManagementCluster bool
	workloadClusterNameCount    int
}

func NewMulticlusterE2ETest(t *testing.T, managementCluster *ClusterE2ETest, workloadClusters ...*ClusterE2ETest) *MulticlusterE2ETest {
	m := &MulticlusterE2ETest{
		T:                    t,
		ManagementCluster:    managementCluster,
		MaxConcurrentWorkers: -1,
	}

	m.WorkloadClusters = make(WorkloadClusters, len(workloadClusters))
	for _, c := range workloadClusters {
		c.clusterFillers = append(c.clusterFillers, api.WithManagementCluster(managementCluster.ClusterName))
		c.ClusterName = m.NewWorkloadClusterName()
		m.WithWorkloadClusters(c)
	}

	return m
}

// WithWorkloadClusters adds ClusterE2ETest's as workload clusters to the test.
func (m *MulticlusterE2ETest) WithWorkloadClusters(workloadClusters ...*ClusterE2ETest) {
	for _, c := range workloadClusters {
		m.WorkloadClusters[c.ClusterName] = &WorkloadCluster{
			ClusterE2ETest:                  c,
			ManagementClusterKubeconfigFile: m.ManagementCluster.kubeconfigFilePath,
		}
	}
}

// NewWorkloadClusterName returns a new unique name for a workload cluster based on the management cluster name.
// This is not thread safe.
func (m *MulticlusterE2ETest) NewWorkloadClusterName() string {
	n := fmt.Sprintf("%s-w-%d", m.ManagementCluster.ClusterName, m.workloadClusterNameCount)
	m.workloadClusterNameCount++
	return n
}

func (m *MulticlusterE2ETest) RunInWorkloadClusters(flow func(*WorkloadCluster)) {
	for name, w := range m.WorkloadClusters {
		m.T.Logf("Running test flow in workload cluster %s", name)
		flow(w)
	}
}

// RunConcurrentlyInWorkloadClusters executes the given flow concurrently for all workload
// clusters. It respects MaxConcurrentWorkers.
func (m *MulticlusterE2ETest) RunConcurrentlyInWorkloadClusters(flow func(*WorkloadCluster)) {
	jobs := make([]func(), 0, len(m.WorkloadClusters))
	for name, wc := range m.WorkloadClusters {
		w := wc
		jobs = append(jobs, func() {
			m.T.Logf("Running test flow in workload cluster %s", name)
			flow(w)
		})
	}
	m.RunConcurrently(jobs...)
}

// RunConcurrently runs the given jobs concurrently using no more than MaxConcurrentWorkers workers.
// If MaxConcurrentWorkers is -1, it will use one worker per job.
func (m *MulticlusterE2ETest) RunConcurrently(flows ...func()) {
	wg := &sync.WaitGroup{}
	workerNum := m.MaxConcurrentWorkers
	if workerNum < 0 {
		workerNum = len(flows)
	}

	jobs := make(chan func())

	wg.Add(workerNum)
	for i := 0; i < workerNum; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				job()
			}
		}()
	}

	for _, flow := range flows {
		jobs <- flow
	}
	close(jobs)

	wg.Wait()
}

func (m *MulticlusterE2ETest) CreateManagementClusterForVersion(eksaVersion string, opts ...CommandOpt) {
	m.ManagementCluster.GenerateClusterConfigForVersion(eksaVersion)
	m.CreateManagementCluster(opts...)
}

// CreateManagementClusterWithConfig first generates a cluster config based on the management cluster test's
// previous configuration and proceeds to create a management cluster with the CLI.
func (m *MulticlusterE2ETest) CreateManagementClusterWithConfig(opts ...CommandOpt) {
	m.ManagementCluster.GenerateClusterConfig()
	m.ManagementCluster.CreateCluster(opts...)
}

func (m *MulticlusterE2ETest) CreateManagementCluster(opts ...CommandOpt) {
	if m.PersistentManagementCluster {
		if _, err := os.Stat(m.ManagementCluster.kubeconfigFilePath()); err == nil {
			m.T.Logf("Persisent cluster: kubeconfig found for management cluster %s, skipping cluster creation", m.ManagementCluster.ClusterName)
			return
		}
	}

	m.ManagementCluster.CreateCluster(opts...)
}

// CreateTinkerbellManagementCluster runs tinkerbell related steps for cluster creation.
func (m *MulticlusterE2ETest) CreateTinkerbellManagementCluster(opts ...CommandOpt) {
	m.ManagementCluster.GenerateClusterConfig()
	m.ManagementCluster.GenerateHardwareConfig()
	m.ManagementCluster.PowerOffHardware()
	m.ManagementCluster.CreateCluster(opts...)
}

func (m *MulticlusterE2ETest) DeleteManagementCluster() {
	m.ManagementCluster.DeleteCluster()
}

// DeleteTinkerbellManagementCluster runs tinkerbell related steps for cluster deletion.
func (m *MulticlusterE2ETest) DeleteTinkerbellManagementCluster() {
	m.ManagementCluster.StopIfFailed()
	m.ManagementCluster.DeleteCluster()
	m.ManagementCluster.ValidateHardwareDecommissioned()
}
