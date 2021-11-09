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

import os
import logging
import argparse

from train import train

from utils import generate_machine_list_file, generate_train_conf_file


logger = logging.getLogger(__name__)


def main(args, extra_args):

    master_addr = os.environ["MASTER_ADDR"]
    master_port = os.environ["MASTER_PORT"]
    worker_addrs = os.environ["WORKER_ADDRS"]
    worker_port = os.environ["WORKER_PORT"]
    world_size = int(os.environ["WORLD_SIZE"])
    rank = int(os.environ["RANK"])

    logger.info(
        "extract cluster info from env variables \n"
        f"master_addr: {master_addr} \n"
        f"master_port: {master_port} \n"
        f"worker_addrs: {worker_addrs} \n"
        f"worker_port: {worker_port} \n"
        f"world_size: {world_size} \n"
        f"rank: {rank} \n"
    )

    if args.job_type == "Predict":
        logging.info("starting the predict job")

    elif args.job_type == "Train":
        logging.info("starting the train job")
        logging.info(f"extra args:\n {extra_args}")
        machine_list_filepath = generate_machine_list_file(
            master_addr, master_port, worker_addrs, worker_port
        )
        logging.info(f"machine list generated in: {machine_list_filepath}")
        local_port = worker_port if rank else master_port
        config_file = generate_train_conf_file(
            machine_list_file=machine_list_filepath,
            world_size=world_size,
            output_model="model.txt",
            local_port=local_port,
            extra_args=extra_args,
        )
        logging.info(f"config generated in: {config_file}")
        train(config_file)
    logging.info("Finish distributed job")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument(
        "--job_type",
        help="Job type to execute",
        choices=["Train", "Predict"],
        required=True,
    )

    logging.basicConfig(format="%(message)s")
    logging.getLogger().setLevel(logging.INFO)
    args, extra_args = parser.parse_known_args()
    main(args, extra_args)
