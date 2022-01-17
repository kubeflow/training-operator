## build the docker file
docker build -f Dockerfile -t merlintang/xgboost-dist-iris:1.0 ./

## push the docker image into docker.io
docker push merlintang/xgboost-dist-iris:1.0

## run the train job
kubectl create -f xgboostjob_v1alpha1_iris_train.yaml

## run the predict job
kubectl create -f xgboostjob_v1alpha1_iris_predict.yaml
