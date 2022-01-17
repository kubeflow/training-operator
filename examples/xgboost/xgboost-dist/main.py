# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import logging

from train import train
from predict import predict
from utils import dump_model


def main(args):

    model_storage_type = args.model_storage_type
    if (model_storage_type == "local" or model_storage_type == "oss"):
      print ( "The storage type is " + model_storage_type)
    else:
      raise Exception("Only supports storage types like local and OSS")

    if args.job_type == "Predict":
        logging.info("starting the predict job")
        predict(args)

    elif args.job_type == "Train":
        logging.info("starting the train job")
        model = train(args)

        if model is not None:
            logging.info("finish the model training, and start to dump model ")
            model_path = args.model_path
            dump_model(model, model_storage_type, model_path, args)

    elif args.job_type == "All":
        logging.info("starting the train and predict job")

    logging.info("Finish distributed XGBoost job")


if __name__ == '__main__':
    parser = argparse.ArgumentParser()

    parser.add_argument(
           '--job_type',
           help="Train, Predict, All",
           required=True
           )
    parser.add_argument(
           '--xgboost_parameter',
           help='XGBoost model parameter like: objective, number_class',
          )
    parser.add_argument(
          '--n_estimators',
          help='Number of trees in the model',
          type=int,
          default=1000
          )
    parser.add_argument(
           '--learning_rate',
           help='Learning rate for the model',
           default=0.1
          )
    parser.add_argument(
          '--early_stopping_rounds',
          help='XGBoost argument for stopping early',
          default=50
          )
    parser.add_argument(
          '--model_path',
          help='place to store model',
          default="/tmp/xgboost_model"
          )
    parser.add_argument(
          '--model_storage_type',
          help='place to store the model',
          default="oss"
          )
    parser.add_argument(
          '--oss_param',
          help='oss parameter if you choose the model storage as OSS type',
          )

    logging.basicConfig(format='%(message)s')
    logging.getLogger().setLevel(logging.INFO)
    main_args = parser.parse_args()
    main(main_args)
