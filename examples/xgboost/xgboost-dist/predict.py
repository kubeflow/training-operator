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

from sklearn.metrics import precision_score

import logging
import numpy as np

from utils import extract_xgbooost_cluster_env, read_predict_data, read_model


def predict(args):
    """
    This is the demonstration for the batch prediction
    :param args: parameter for model related config
    """

    addr, port, rank, world_size = extract_xgbooost_cluster_env()

    dmatrix, y_test = read_predict_data(rank, world_size, None)

    model_path = args.model_path
    storage_type = args.model_storage_type
    booster = read_model(storage_type, model_path, args)

    preds = booster.predict(dmatrix)

    best_preds = np.asarray([np.argmax(line) for line in preds])
    score = precision_score(y_test, best_preds, average='macro')

    logging.info("Predict accuracy: %f", score)
