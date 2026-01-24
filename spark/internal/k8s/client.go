package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is a Kubernetes client for managing spark resources.
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient creates a new Kubernetes client.
func NewClient() (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

// CreateSpark creates all Kubernetes resources for a new spark.
func (c *Client) CreateSpark(ctx context.Context, resources *SparkResources) error {
	// Create ConfigMap
	_, err := c.clientset.CoreV1().ConfigMaps(SparkNamespace).Create(ctx, resources.CreateConfigMap(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create configmap: %w", err)
	}

	// Create Secret
	_, err = c.clientset.CoreV1().Secrets(SparkNamespace).Create(ctx, resources.CreateSecret(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	// Create PVC
	_, err = c.clientset.CoreV1().PersistentVolumeClaims(SparkNamespace).Create(ctx, resources.CreatePVC(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pvc: %w", err)
	}

	// Create Service
	_, err = c.clientset.CoreV1().Services(SparkNamespace).Create(ctx, resources.CreateService(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Create Deployment
	_, err = c.clientset.AppsV1().Deployments(SparkNamespace).Create(ctx, resources.CreateDeployment(), metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

// ListSparks lists all active sparks in the cluster.
func (c *Client) ListSparks(ctx context.Context) ([]string, error) {
	deployments, err := c.clientset.AppsV1().Deployments(SparkNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=spark",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var sparks []string
	for _, deployment := range deployments.Items {
		if name, ok := deployment.Labels["spark-name"]; ok {
			sparks = append(sparks, name)
		}
	}

	return sparks, nil
}

// DeleteSpark deletes all Kubernetes resources associated with a spark.
func (c *Client) DeleteSpark(ctx context.Context, name string) error {
	// Delete deployment (cascades to pods)
	err := c.clientset.AppsV1().Deployments(SparkNamespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete service
	err = c.clientset.CoreV1().Services(SparkNamespace).Delete(ctx, name+"-ssh", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Delete PVC
	err = c.clientset.CoreV1().PersistentVolumeClaims(SparkNamespace).Delete(ctx, name+"-storage", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pvc: %w", err)
	}

	// Delete Secret
	err = c.clientset.CoreV1().Secrets(SparkNamespace).Delete(ctx, name+"-secret", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Delete ConfigMap
	err = c.clientset.CoreV1().ConfigMaps(SparkNamespace).Delete(ctx, name+"-config", metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete configmap: %w", err)
	}

	return nil
}

// GetSparkPod retrieves the running pod for a given spark.
func (c *Client) GetSparkPod(ctx context.Context, name string) (*corev1.Pod, error) {
	pods, err := c.clientset.CoreV1().Pods(SparkNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=spark,spark-name=" + name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pod found for spark %s", name)
	}

	// Return the first running pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("no running pod found for spark %s", name)
}

// GetDeployment retrieves the deployment for a given spark.
func (c *Client) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(SparkNamespace).Get(ctx, name, metav1.GetOptions{})
}
