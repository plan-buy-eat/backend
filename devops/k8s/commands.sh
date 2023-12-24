# https://www.linuxtechi.com/install-kubernetes-on-ubuntu-22-04/#8_Install_Calico_Network_Plugin

sudo hostnamectl set-hostname "k8smaster.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8sworker1.cluster-prod.turevskiy.kharkiv.ua"
sudo hostnamectl set-hostname "k8sworker2.cluster-prod.turevskiy.kharkiv.ua"

192.168.0.159   k8smaster.cluster-prod.turevskiy.kharkiv.ua k8smaster
192.168.0.170   k8sworker1.cluster-prod.turevskiy.kharkiv.ua k8sworker1
192.168.0.236   k8sworker2.cluster-prod.turevskiy.kharkiv.ua k8sworker2
192.168.0.237 k8sworkera.cluster-prod.turevskiy.kharkiv.ua k8sworkera

sudo kubeadm init --control-plane-endpoint=k8smaster.cluster-prod.turevskiy.kharkiv.ua


mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config