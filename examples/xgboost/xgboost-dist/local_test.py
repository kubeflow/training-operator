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

"""
this file contains tests for xgboost local train and predict in single machine.
Note: this is not for distributed train and predict test
"""
from utils import dump_model, read_model, read_train_data, read_predict_data
import xgboost as xgb
import logging
import numpy as np
from sklearn.metrics import precision_score

logger = logging.getLogger(__name__)


def test_train_model():
    """
    test xgboost train in a single machine
    :return: trained model
    """
    rank = 1
    world_size = 10
    place = "/tmp/data"
    dmatrix = read_train_data(rank, world_size, place)

    param_xgboost_default = {'max_depth': 2, 'eta': 1, 'silent': 1,
                             'objective': 'multi:softprob', 'num_class': 3}

    booster = xgb.train(param_xgboost_default, dtrain=dmatrix)

    assert booster is not None

    return booster


def test_model_predict(booster):
    """
    test xgboost train in the single node
    :return: true if pass the test
    """
    rank = 1
    world_size = 10
    place = "/tmp/data"
    dmatrix, y_test = read_predict_data(rank, world_size, place)

    preds = booster.predict(dmatrix)
    best_preds = np.asarray([np.argmax(line) for line in preds])
    score = precision_score(y_test, best_preds, average='macro')

    assert score > 0.99

    logging.info("Predict accuracy: %f", score)

    return True


def test_upload_model(model, model_path, args):

    return dump_model(model, type="local", model_path=model_path, args=args)


def test_download_model(model_path, args):

    return read_model(type="local", model_path=model_path, args=args)


def run_test():
    args = {}
    model_path = "/tmp/xgboost"

    logging.info("Start the local test")

    booster = test_train_model()
    test_upload_model(booster, model_path, args)
    booster_new = test_download_model(model_path, args)
    test_model_predict(booster_new)

    logging.info("Finish the local test")


if __name__ == '__main__':

    logging.basicConfig(format='%(message)s')
    logging.getLogger().setLevel(logging.INFO)

    run_test()
