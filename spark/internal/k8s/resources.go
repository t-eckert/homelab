package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SparkResources struct {
	Name             string
	GitRepo          string
	DatabaseURL      string
	AnthropicAPIKey  string
	SSHPublicKey     string
	GitHubToken      string
}

const SparkNamespace = "spark"

func (s *SparkResources) CreateConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name + "-config",
			Namespace: SparkNamespace,
			Labels: map[string]string{
				"app":        "spark",
				"spark-name": s.Name,
			},
		},
		Data: map[string]string{
			"authorized_keys": s.SSHPublicKey,
			"git_repo":        s.GitRepo,
		},
	}
}

func (s *SparkResources) CreateSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name + "-secret",
			Namespace: SparkNamespace,
			Labels: map[string]string{
				"app":        "spark",
				"spark-name": s.Name,
			},
		},
		StringData: map[string]string{
			"DATABASE_URL":      s.DatabaseURL,
			"ANTHROPIC_API_KEY": s.AnthropicAPIKey,
			"GITHUB_TOKEN":      s.GitHubToken,
		},
	}
}

func (s *SparkResources) CreatePVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name + "-storage",
			Namespace: SparkNamespace,
			Labels: map[string]string{
				"app":        "spark",
				"spark-name": s.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}
}

func (s *SparkResources) CreateService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name + "-ssh",
			Namespace: SparkNamespace,
			Labels: map[string]string{
				"app":        "spark",
				"spark-name": s.Name,
			},
			Annotations: map[string]string{
				"tailscale.com/hostname": "spark-" + s.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:              corev1.ServiceTypeLoadBalancer,
			LoadBalancerClass: stringPtr("tailscale"),
			Ports: []corev1.ServicePort{
				{
					Name:     "ssh",
					Port:     22,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app":        "spark",
				"spark-name": s.Name,
			},
		},
	}
}

func (s *SparkResources) CreateDeployment() *appsv1.Deployment {
	replicas := int32(1)
	runAsUser := int64(0)
	fsGroup := int64(1000)

	initScript := s.buildInitScript()

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: SparkNamespace,
			Labels: map[string]string{
				"app":        "spark",
				"spark-name": s.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":        "spark",
					"spark-name": s.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":        "spark",
						"spark-name": s.Name,
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: &fsGroup,
					},
					Containers: []corev1.Container{
						{
							Name:  "debian",
							Image: "debian:bookworm",
							Command: []string{
								"/bin/bash",
								"-c",
								initScript,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "ssh",
									ContainerPort: 22,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "DATABASE_URL",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: s.Name + "-secret",
											},
											Key: "DATABASE_URL",
										},
									},
								},
								{
									Name: "ANTHROPIC_API_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: s.Name + "-secret",
											},
											Key: "ANTHROPIC_API_KEY",
										},
									},
								},
								{
									Name:  "SPARK_NAME",
									Value: s.Name,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "spark-storage",
									MountPath: "/home/user",
								},
								{
									Name:      "spark-config",
									MountPath: "/tmp/spark-config",
									ReadOnly:  true,
								},
								{
									Name:      "spark-secret",
									MountPath: "/tmp/spark-secret",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:              &runAsUser,
								AllowPrivilegeEscalation: boolPtr(true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SETUID",
										"SETGID",
									},
								},
								ReadOnlyRootFilesystem: boolPtr(false),
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "spark-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: s.Name + "-storage",
								},
							},
						},
						{
							Name: "spark-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: s.Name + "-config",
									},
								},
							},
						},
						{
							Name: "spark-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: s.Name + "-secret",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (s *SparkResources) buildInitScript() string {
	script := `#!/bin/bash
set -e

# Install dependencies
apt-get update && apt-get install -y \
    openssh-server \
    sudo \
    curl \
    git \
    wget \
    vim \
    tmux \
    build-essential \
    ca-certificates

# Create user with sudo access
useradd -u 1000 -d /home/user -s /bin/bash user || true
echo "user ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/user
chmod 440 /etc/sudoers.d/user

# Create user home directory structure
mkdir -p /home/user/.ssh /home/user/.local/bin /home/user/.config

# Copy authorized keys from ConfigMap
cp /tmp/spark-config/authorized_keys /home/user/.ssh/authorized_keys
chmod 600 /home/user/.ssh/authorized_keys
chmod 700 /home/user/.ssh

# Configure GitHub CLI authentication
mkdir -p /home/user/.config/gh
if [ -f /tmp/spark-secret/GITHUB_TOKEN ]; then
    echo "github.com:" > /home/user/.config/gh/hosts.yml
    echo "    user: t-eckert" >> /home/user/.config/gh/hosts.yml
    echo "    oauth_token: $(cat /tmp/spark-secret/GITHUB_TOKEN)" >> /home/user/.config/gh/hosts.yml
    echo "    git_protocol: https" >> /home/user/.config/gh/hosts.yml
    chmod 700 /home/user/.config/gh
    chmod 600 /home/user/.config/gh/hosts.yml
fi

# Install Claude Code CLI as user
su - user -c "curl -fsSL https://raw.githubusercontent.com/anthropics/claude-code/main/install.sh | sh"

# Clone dotfiles if not already present
if [ ! -d /home/user/.dotfiles ]; then
    su - user -c "git clone https://github.com/t-eckert/dotfiles.git /home/user/.dotfiles"
    su - user -c "cd /home/user/.dotfiles && ./install.sh" || true
fi
`

	if s.GitRepo != "" {
		script += `
# Clone user's git repository
REPO_URL="` + s.GitRepo + `"
if [ -n "$REPO_URL" ] && [ ! -d /home/user/project ]; then
    su - user -c "git clone $REPO_URL /home/user/project"
fi
`
	}

	script += `
# Set ownership
chown -R 1000:1000 /home/user

# Configure SSH
mkdir -p /run/sshd
ssh-keygen -A

# Configure sshd_config
cat >> /etc/ssh/sshd_config <<EOF
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AllowUsers user
EOF

# Start SSH daemon
/usr/sbin/sshd -D
`

	return script
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
