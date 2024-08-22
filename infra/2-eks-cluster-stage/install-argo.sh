#!/bin/bash

echo "---> installing helm"
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh
echo "---> getting kubeconfig"
aws eks update-kubeconfig --name $1 --region $2
argoInstalled=$(helm list -n argocd --filter argocd --output json | jq -r '.[0].status')
echo $argoInstalled
if [[ "$argoInstalled" == "deployed" ]]; then
    echo "---> argoCD already installed"
elif [[ "$argoInstalled" == "null" ]]
    echo "---> argoCD no Installed"
    echo "---> installing argo"
    helm repo add argocd https://argoproj.github.io/argo-helm
    helm repo update
    helm install argocd argocd/argo-cd --namespace argocd --create-namespace  -f ./k8-manifests/helm-argo-installation/argocd.yaml
else
    echo "check Helm chart status"
fi
