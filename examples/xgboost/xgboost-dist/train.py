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


import logging
import xgboost as xgb
import traceback

from tracker import RabitTracker
from utils import read_train_data, extract_xgbooost_cluster_env

logger = logging.getLogger(__name__)


def train(args):
    """
    :param args: configuration for train job
    :return: XGBoost model
    """
    addr, port, rank, world_size = extract_xgbooost_cluster_env()
    rabit_tracker = None

    try:
        """start to build the network"""
        if world_size > 1:
            if rank == 0:
                logger.info("start the master node")

                rabit = RabitTracker(hostIP="0.0.0.0", nslave=world_size,
                                     port=port, port_end=port + 1)
                rabit.start(world_size)
                rabit_tracker = rabit
                logger.info('###### RabitTracker Setup Finished ######')

            envs = [
                'DMLC_NUM_WORKER=%d' % world_size,
                'DMLC_TRACKER_URI=%s' % addr,
                'DMLC_TRACKER_PORT=%d' % port,
                'DMLC_TASK_ID=%d' % rank
            ]
            logger.info('##### Rabit rank setup with below envs #####')
            for i, env in enumerate(envs):
                logger.info(env)
                envs[i] = str.encode(env)

            xgb.rabit.init(envs)
            logger.info('##### Rabit rank = %d' % xgb.rabit.get_rank())
            rank = xgb.rabit.get_rank()

        else:
            world_size = 1
            logging.info("Start the train in a single node")

        df = read_train_data(rank=rank, num_workers=world_size, path=None)
        kwargs = {}
        kwargs["dtrain"] = df
        kwargs["num_boost_round"] = int(args.n_estimators)
        param_xgboost_default = {'max_depth': 2, 'eta': 1, 'silent': 1,
                                 'objective': 'multi:softprob', 'num_class': 3}
        kwargs["params"] = param_xgboost_default

        logging.info("starting to train xgboost at node with rank %d", rank)
        bst = xgb.train(**kwargs)

        if rank == 0:
            model = bst
        else:
            model = None

        logging.info("finish xgboost training at node with rank %d", rank)

    except Exception as e:
        logger.error("something wrong happen: %s", traceback.format_exc())
        raise e
    finally:
        logger.info("xgboost training job finished!")
        if world_size > 1:
            xgb.rabit.finalize()
        if rabit_tracker:
            rabit_tracker.join()

    return model
