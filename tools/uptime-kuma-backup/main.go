package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	defaultNamespace     = "uptime-kuma"
	defaultPodName       = "uptime-kuma-0"
	defaultContainerName = "uptime-kuma"
	defaultSourcePath    = "/app/data"
	defaultStorageClass  = "local-path"
)

type BackupConfig struct {
	Namespace     string
	PodName       string
	ContainerName string
	SourcePath    string
	BackupSize    string
	StorageClass  string
	BackupName    string
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	namespace := flag.String("namespace", defaultNamespace, "Namespace where Uptime Kuma is running")
	podName := flag.String("pod", defaultPodName, "Name of the Uptime Kuma pod")
	containerName := flag.String("container", defaultContainerName, "Name of the container in the pod")
	sourcePath := flag.String("source", defaultSourcePath, "Path to backup from inside the container")
	backupSize := flag.String("size", "5Gi", "Size of the backup PVC")
	storageClass := flag.String("storage-class", defaultStorageClass, "Storage class for the backup PVC")
	backupName := flag.String("name", fmt.Sprintf("uptime-kuma-backup-%s", time.Now().Format("20060102-150405")), "Name for the backup PVC")

	flag.Parse()

	config := &BackupConfig{
		Namespace:     *namespace,
		PodName:       *podName,
		ContainerName: *containerName,
		SourcePath:    *sourcePath,
		BackupSize:    *backupSize,
		StorageClass:  *storageClass,
		BackupName:    *backupName,
	}

	// Build Kubernetes config
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	ctx := context.Background()

	// Step 1: Create backup PVC
	log.Printf("Creating backup PVC: %s", config.BackupName)
	pvc, err := createBackupPVC(ctx, clientset, config)
	if err != nil {
		log.Fatalf("Error creating backup PVC: %v", err)
	}
	log.Printf("✓ Created PVC: %s", pvc.Name)

	// Step 2: Create backup job
	log.Printf("Creating backup job")
	job, err := createBackupJob(ctx, clientset, config)
	if err != nil {
		log.Fatalf("Error creating backup job: %v", err)
	}
	log.Printf("✓ Created backup job: %s", job.Name)

	// Step 3: Wait for job completion
	log.Printf("Waiting for backup to complete...")
	if err := waitForJobCompletion(ctx, clientset, config.Namespace, job.Name); err != nil {
		log.Fatalf("Error waiting for job completion: %v", err)
	}

	log.Printf("✓ Backup completed successfully!")
	log.Printf("\nBackup details:")
	log.Printf("  PVC Name: %s", config.BackupName)
	log.Printf("  Namespace: %s", config.Namespace)
	log.Printf("  Size: %s", config.BackupSize)
	log.Printf("\nTo restore from this backup, you can mount the PVC '%s' to a pod.", config.BackupName)
}

func createBackupPVC(ctx context.Context, clientset *kubernetes.Clientset, config *BackupConfig) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.BackupName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				"app":     "uptime-kuma-backup",
				"created": time.Now().Format("2006-01-02"),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: &config.StorageClass,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(config.BackupSize),
				},
			},
		},
	}

	return clientset.CoreV1().PersistentVolumeClaims(config.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
}

func createBackupJob(ctx context.Context, clientset *kubernetes.Clientset, config *BackupConfig) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-job", config.BackupName)
	backoffLimit := int32(3)
	ttlSecondsAfterFinished := int32(3600) // Keep job for 1 hour after completion

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				"app": "uptime-kuma-backup",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "uptime-kuma",
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "backup",
							Image: "bitnami/kubectl:latest",
							Command: []string{
								"/bin/sh",
								"-c",
								fmt.Sprintf(`
set -e
echo "Starting backup process..."
echo "Source pod: %s"
echo "Source path: %s"
echo "Backup destination: /backup"

# Copy data from the source pod to the backup volume
kubectl exec -n %s %s -c %s -- tar czf - -C %s . | tar xzf - -C /backup

echo "Backup completed successfully!"
ls -lah /backup
								`,
									config.PodName,
									config.SourcePath,
									config.Namespace,
									config.PodName,
									config.ContainerName,
									config.SourcePath,
								),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "backup",
									MountPath: "/backup",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "backup",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: config.BackupName,
								},
							},
						},
					},
				},
			},
		},
	}

	return clientset.BatchV1().Jobs(config.Namespace).Create(ctx, job, metav1.CreateOptions{})
}

func waitForJobCompletion(ctx context.Context, clientset *kubernetes.Clientset, namespace, jobName string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for job to complete")
		case <-ticker.C:
			job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("error getting job status: %v", err)
			}

			if job.Status.Succeeded > 0 {
				return nil
			}

			if job.Status.Failed > 0 {
				// Get pod logs for debugging
				pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("job-name=%s", jobName),
				})
				if err == nil && len(pods.Items) > 0 {
					podName := pods.Items[0].Name
					fmt.Fprintf(os.Stderr, "\nJob failed. Check logs with: kubectl logs -n %s %s\n", namespace, podName)
				}
				return fmt.Errorf("job failed")
			}

			fmt.Print(".")
		}
	}
}
