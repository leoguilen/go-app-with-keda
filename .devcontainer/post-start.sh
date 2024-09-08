k3d cluster create --registry-create k3d-registry:0.0.0.0:5000
echo '127.0.0.1 k3d-registry' | sudo tee -a /etc/hosts

kubectl apply --server-side -f https://github.com/kedacore/keda/releases/download/v2.15.1/keda-2.15.1.yaml

docker buildx build --build-arg="APP_NAME=consumer" -t k3d-registry:5000/consumer --push .
docker buildx build --build-arg="APP_NAME=producer" -t k3d-registry:5000/producer --push .