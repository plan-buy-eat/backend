# https://www.linuxtechi.com/install-kubernetes-on-ubuntu-22-04/#8_Install_Calico_Network_Plugin

sudo hostnamectl set-hostname "k8smaster.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8sworker1.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8sworker2.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8smaster2.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8sworker3.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8sworker4.cluster-prod.turevskiy.kharkiv.ua"

# /etc/hosts
192.168.0.223   k8smaster.cluster-prod.turevskiy.kharkiv.ua k8smaster
192.168.0.149   k8sworker1.cluster-prod.turevskiy.kharkiv.ua k8sworker1
192.168.0.54    k8sworker2.cluster-prod.turevskiy.kharkiv.ua k8sworker2
192.168.0.237 k8sworkera.cluster-prod.turevskiy.kharkiv.ua k8sworkera

# all
sudo swapoff -a
sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

sudo tee /etc/modules-load.d/containerd.conf <<EOF
overlay
br_netfilter
EOF
sudo modprobe overlay
sudo modprobe br_netfilter

sudo tee /etc/sysctl.d/kubernetes.conf <<EOT
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
EOT

sudo sysctl --system

sudo apt install -y curl gnupg2 software-properties-common apt-transport-https ca-certificates

sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmour -o /etc/apt/trusted.gpg.d/docker.gpg
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"

sudo apt update
sudo apt install -y containerd.io

containerd config default | sudo tee /etc/containerd/config.toml >/dev/null 2>&1
sudo sed -i 's/SystemdCgroup \= false/SystemdCgroup \= true/g' /etc/containerd/config.toml

sudo systemctl restart containerd
sudo systemctl enable containerd

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.28/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.28/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list

sudo apt update
sudo apt install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl

# master only
sudo kubeadm init --control-plane-endpoint=k8smaster.cluster-prod.turevskiy.kharkiv.ua

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

kubectl cluster-info
kubectl get nodes

# workers only
$ kubectl cluster-info
$ kubectl get nodes

# master 
kubectl get nodes

kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.0/manifests/calico.yaml
kubectl get pods -n kube-system

kubectl get nodes